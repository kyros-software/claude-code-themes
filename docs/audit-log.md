> **Documento histórico.** Esto audita la implementación en **Python**, que fue
> la del proyecto hasta la versión 2.0.0. El runtime es ahora un binario de Go y
> los números de aquí ya no describen lo que corre en tu máquina: la statusline
> pasó de 22,4 ms a 3,5 y el hook de 21,3 a 1,6. Se conserva porque las
> mediciones y el razonamiento siguen siendo ciertos sobre lo que medían, y
> porque explican **por qué** se acabó cambiando de lenguaje: el código propio
> costaba 1,5 ms y el resto era Python presentándose. Los nombres de fichero e
> identificador son los de entonces (`statusline.sh`, `ESTADOS`).

# Auditoría de la statusline

Repaso de rendimiento y errores de `statusline.sh`, medido sobre el commit
`e94ac98` el 1 de septiembre de 2026. Python 3.12.3, WSL2, git 2.43.

Todo lo que hay aquí está **medido, no estimado**. Lo que no pude reproducir está
en [Descartado](#descartado) para que nadie lo vuelva a perseguir.

> **Estado a 1 de septiembre de 2026.** Los bugs 1-4 están **arreglados** y el
> `python3 -S` **aplicado**. Los puntos de diseño 5 y 6 también están cerrados
> ya: ver «Los dos puntos de diseño» al final. Las medidas de más abajo
> son las de *antes* de arreglar nada: son la línea base, y las dejo tal cual
> para que se pueda comparar. Al final hay una sección con las de después.

## Cómo reproducir las medidas

```bash
J='{"session_id":"bench","model":{"display_name":"Opus 5 (1M context)"},
    "workspace":{"current_dir":"'$PWD'"},"effort":{"level":"xhigh"},
    "cost":{"total_cost_usd":28.29,"total_lines_added":184,"total_lines_removed":37,
            "total_duration_ms":4320000},
    "context_window":{"used_percentage":36,"context_window_size":1000000},
    "prompt_cache":{"hit_ratio":0.98},
    "rate_limits":{"five_hour":{"used_percentage":41},"seven_day":{"used_percentage":13}}}'

# tiempo por invocación
t0=$(date +%s%N); for i in $(seq 30); do echo "$J" | COLUMNS=116 ./statusline.sh >/dev/null; done
t1=$(date +%s%N); echo $(( (t1-t0)/30/1000000 )) ms

# desglose de los imports
python3 -X importtime -c 'import sys,json,os,re,subprocess,time' 2>&1 | sort -t'|' -k2 -rn | head
```

---

## Rendimiento

### A dónde se va el tiempo

| Componente | Coste | % |
| --- | --- | --- |
| **Total por invocación** (este repo) | **25,3 ms** | 100% |
| ↳ arranque de CPython (`python3 -c pass`) | 11,3 ms | 45% |
| ↳ ↳ *de los cuales el módulo `site`* | *4,7 ms* | *19%* |
| ↳ imports: `json` 5,1 · `re` 4,0 · `subprocess` 3,8 (+`threading`, `selectors`) | ~6 ms | 24% |
| ↳ compilar los 10 KB de `PY_SRC` | 2,0 ms | 8% |
| ↳ los dos `git` | 2,0 ms | 8% |
| ↳ `$(cat <<EOF)` + subshell de bash | 0,8 ms | 3% |

En un repo grande (`avanzapi-frontend`, ~840 ficheros) sube a **30,7 ms**, porque
`git status --porcelain` pasa de 1,1 a 6,0 ms.

**El cuello no es git: es el arranque del intérprete.** El 70% del tiempo se va
antes de ejecutar la primera línea útil. Optimizar las llamadas a git es pulir
el 8% del problema.

### Frecuencia real

`settings.json` declara `refreshInterval: 1`, pero la medida no da 1 Hz.
Muestreando los mtime de `/tmp/claude-statusline-*` a 20 Hz durante 15 s:

```
sesiones activas: 3 de 12
por sesión:       ~0,7 Hz
agregado:         2,2 invocaciones/s  ->  ~6% de un núcleo, continuo
```

No es un incendio, pero es permanente mientras la CLI esté abierta, y escala con
el número de sesiones abiertas a la vez.

### Optimizaciones prototipadas

Ambas verificadas produciendo salida **byte a byte idéntica** al original.

| Cambio | Resultado | Veredicto |
| --- | --- | --- |
| `exec python3 -S -c` (saltarse `site`) | 25,3 → **23,3 ms** (−8%) | **Aplicar.** Riesgo cero: el script solo usa stdlib. |
| \+ mover git a bash y quitar `subprocess` | → **21,1 ms** (−17%) | **No aplicar.** Ver abajo. |

El segundo suena bien y no lo vale: bash no tiene el `cwd`, que viene en el JSON
por stdin y stdin solo se lee una vez. Habría que fiarse de `$PWD` asumiendo que
Claude Code invoca la statusline dentro del workspace — una suposición no
documentada, a cambio de 2 ms.

**La palanca de verdad no es micro-optimizar, es `refreshInterval`.** Lo único
que exige refrescar cada segundo es la animación de las patas. A `refreshInterval: 5`
el coste cae cinco veces.

---

## Bugs

| # | Severidad | Qué pasa | Estado |
| --- | --- | --- | --- |
| 1 | **Alta** | Un `used_percentage` no numérico deja la statusline en blanco | **arreglado** — todo número del JSON pasa por `num()` |
| 2 | Media | Reescribe el fichero de estado en cada refresco aunque no cambie | **arreglado** — solo escribe cuando cambia |
| 3 | Baja | Ficheros huérfanos en `/tmp`, uno por sesión, nunca se limpian | **arreglado** — barre los de más de un día, y `SessionEnd` borra el suyo |
| 4 | Cosmética | Descriptor de fichero que no se cierra | **arreglado** — `finally: os.close(fd)` |

### 1 · Crash con `used_percentage` no numérico

**Alta**, porque el fallo no degrada: desaparece la statusline entera.

```
$ echo '{"context_window":{"used_percentage":"abc"}}' | ./statusline.sh
Traceback (most recent call last):
  File "<string>", line 194, in <module>
ValueError: cannot convert float NaN to integer
```

| Valor | Resultado |
| --- | --- |
| `"NaN"`, `"abc"`, `[]` | **crash**, salida vacía, exit 1 |
| `"36"`, `true`, `-5`, `150` | ok |

La causa es que `float(pct)` y `round(float(pct))` en la banda 1 son **el único
punto del script sin `try/except`**. Todos los demás campos numéricos —`cache`,
`coste`, `ctxsz`, `durms`, `rl5`, `rl7`— sí están guardados. Es una
inconsistencia, no una decisión.

Arreglo: normalizar `pct` a `float` o `None` una sola vez, junto al resto de
lecturas, y dejar de reconvertir en cada uso.

### 2 · Escritura redundante del estado

El fichero por sesión guarda la etiqueta anterior para poner en negrita un
refresco al cruzar umbral. Se escribe **siempre**, cambie o no:

```
intento 1: mtime=10:53:55.511667035  contenido=8 bytes
intento 2: mtime=10:53:55.538067034  contenido=8 bytes   <- mismo contenido
intento 3: mtime=10:53:55.564467033  contenido=8 bytes   <- mismo contenido
```

Un `write()` más metadatos de sistema de ficheros por invocación, por sesión, para
no cambiar nada. El bloque ya lee el valor anterior: basta con escribir solo si
difiere.

### 3 · Basura en `/tmp`

Un fichero por `session_id`, creado siempre, borrado nunca. En la máquina de
pruebas: **12 ficheros, 9 de sesiones muertas**. Son 8 bytes cada uno, así que el
problema no es el espacio sino que crece sin techo y `/tmp` no siempre se limpia
al reiniciar (WSL entre ellos).

Arreglo: al escribir, barrer los `claude-statusline-*` con mtime de más de un día.

### 4 · Descriptor sin cerrar

```python
return os.get_terminal_size(os.open("/dev/tty", os.O_RDONLY)).columns
```

`os.open` devuelve un fd que nadie cierra. Inocuo —el proceso muere acto
seguido— y solo se ejecuta en la rama de respaldo, cuando falta `COLUMNS`. Está
aquí por higiene, no por impacto.

---

## Diseño

No son errores: son decisiones que conviene tomar a sabiendas.

### 5 · El k.o. es prácticamente inalcanzable

Con los tres consumos presentes, el estado `k.o.` exige `uso > 99,999`, y siendo
una media ponderada eso significa los tres al 100% clavado:

| ctx | 5h | 7d | uso | estado |
| --- | --- | --- | --- | --- |
| 100 | 100 | 100 | 100,000 | k.o. |
| 100 | 100 | 99 | 99,800 | ahogada |
| 100 | 90 | 90 | 95,000 | ahogada |
| 100 | — | — | 100,000 | k.o. |

Es coherente con lo que documenta [VIDA.md](design/vitals.md), y aun así merece decirlo
claro: **el sprite del k.o., el que más trabajo llevó, no se va a ver casi
nunca.** Solo aparece si el único dato disponible es el contexto. Si se quiere
que sea alcanzable, el umbral tiene que bajar (p. ej. 97) o el k.o. debe
dispararse por el máximo de los tres en vez de por la media.

### 6 · La animación va a tope por defecto

El código leía una variable de entorno de calma y, solo si estaba puesta,
limitaba el paso a cuatro refrescos de cada doce:

```python
anda = bool(E.get("anda")) and (paso % 12 < 4 if _calma else True)
```

Sin esa variable las patas alternan en **cada** refresco, para
siempre, en visión periférica. El modo calmado —andar 4 segundos de cada 12— es
mejor default; quien quiera el baile continuo que lo pida con una variable.

---

## Descartado

Comprobado y **no** es problema. Documentado para no repetir el trabajo.

- **Inyección de shell.** No hay. El JSON nunca pasa por el shell y `git -C` se
  invoca con lista de argumentos, sin `shell=True`. Probado con
  `display_name: "'; rm -rf /"` → se pinta literal.
- **Caracteres de doble ancho descuadrando la mascota.** No hay ninguno. Todos los
  glifos no-ASCII del script son East Asian *Ambiguous*, que se pintan a una
  columna; cero `W`, cero `F`, cero Nerd Font. `vis()` cuenta bien.
- **Contención de `index.lock` por lanzar `git status` en bucle.** No ocurre: el
  mtime de `.git/index` no cambia tras el `status`. `git --no-optional-locks status`
  sería profiláctico. Ojo con la sintaxis: la opción va **antes** del subcomando,
  ponerla detrás es error.
- **`assemble()` es O(n²).** Lo es, y da igual: n ≤ 6.
- **Robustez de entrada.** JSON inválido, `{}`, `cwd` inexistente, `COLUMNS` de 20
  a 200: todo degrada limpio, exit 0, sin wrap ni desbordes.

---

## Orden sugerido

1. Bug 1 — es el único que rompe algo visible.
2. `python3 -S` — 8% gratis.
3. Bugs 2 y 3 — higiene, cinco minutos.
4. Diseño 6 — invertir el default de la animación.
5. Diseño 5 y bug 4 — cuando apetezca.

---

Ver también el [README](../README.md) para las bandas y la paleta, y
[VIDA.md](design/vitals.md) para la fórmula del estado de la mascota.

---

## Después de arreglarlo

Mismas condiciones, mismo repo, mismo JSON de prueba.

| | antes | después |
| --- | --- | --- |
| Tiempo por invocación | 25,3 ms | **24,9 ms** |
| `used_percentage` no numérico | crash, salida vacía | degrada, exit 0 |
| Escrituras de estado por refresco | 1 siempre | 0 salvo cambio |
| Huérfanos en `/tmp` | crecen sin techo | se barren a las 24 h |

El tiempo baja **solo 0,4 ms** y eso merece explicación: `python3 -S` quita
2,0 ms, pero el sistema de evoluciones añade la lectura de `~/.claude/pet.json`
y el import del módulo. La cuenta neta es que **todo el tamagotchi entró
gratis**, no que el arreglo no sirviera.

Dos decisiones de diseño salieron de esta auditoría:

- **El dibujo vive en su propio módulo, no incrustado en el `.sh`.** Un módulo
  importado usa caché de bytecode; un `python3 -c` recompila su fuente en cada
  refresco. Eso devuelve los 2 ms del `compile()` que medía la tabla de arriba.
- **`tempfile` se importa dentro de `escribir_pet()`**, no arriba. Cuesta 2,0 ms
  y la statusline lee ese fichero en cada refresco pero **no lo escribe nunca**:
  solo escriben los hooks y `/feed`.

Y una que no cambió: `git --no-optional-locks` **sí** está aplicado, aunque la
auditoría lo clasificara como profiláctico. Es gratis y el escenario que evita
—dos sesiones peleándose por `index.lock`— es real aunque no lo reprodujera.

---

## Segunda ronda: la revisión de las evoluciones

Un `/code-review` sobre el commit de las evoluciones sacó **quince hallazgos, los
quince reales**. Reproduje los tres peores antes de tocar nada. Todos arreglados.

### El grave

**Ejecución de código desde cualquier repo que abras.** `python3 -c` mete el
directorio actual en `sys.path` como `""`, y la statusline corre con el cwd
puesto en tu proyecto. `sys.path.insert(0, SL_DIR)` empujaba el cwd a la
posición 1 en vez de quitarlo, así que **si faltaba ese módulo en `~/.claude` —el
camino de degradación que el propio README anuncia— se importaba el del
del repo abierto**, ejecutándolo una vez por refresco, con la excepción tragada
por el `try` del import. Reproducido: `*** CODIGO DEL REPO EJECUTADO ***`, rc=0,
sin rastro. Ahora el cwd se purga de `sys.path` antes de importar.

### El vergonzoso

**El bug 1 de la primera ronda, reintroducido en dos ficheros nuevos.** El
`num()` que blinda el JSON de stdin no se aplicó ni a `~/.claude/pet.json` ni al
fichero de sesión de `/tmp`. Un `{"hambre":"mucha"}` volvía a dejar la statusline
en blanco. Los dos ficheros son editables por cualquiera y uno vive en `/tmp`.
Ahora todo campo de los dos pasa por un normalizador de tipos.

### Los otros trece

| Qué | Cómo se veía |
| --- | --- |
| `dict(PET_VACIO)` era copia superficial | `contadores` aliasaba el dict del módulo: un `contar()` contaminaba todas las lecturas siguientes del proceso |
| `sesiones_ctx100` contado dos veces | el kraken se alcanzaba en 2 sesiones en vez de 3 |
| `t0` ausente = epoch 0 | sesiones de 56 años que regalaban `buey` |
| `_subio` filtrado por `leer_pet` | el bocadillo de subida de nivel era código muerto |
| tope diario de `/feed` sobre `hoy[-40:]` | se saltaba en cuanto rotaba el registro |
| `git commit` sin anclar | un `grep "git commit"` daba +12 xp |
| `\bok\b` con `re.I` | cualquier salida que dijera "ok" daba +15 xp |
| `session_id` sin validar en un `open()` | travesía de ruta fuera de `TMPDIR` |
| marcadores `claude-pet-todos-*` | prefijo que el barrido de huérfanos no alcanzaba |
| `json.load` sin `try` en el desinstalador | con `set -e`, un settings.json roto impedía desinstalar |
| `settings.json` escrito sin átomo | un fallo a media escritura vaciaba tu configuración global |
| `alimentar(ahora=…)` a medias | `dia` del reloj real y `ayer` del parámetro |
| `fenix` y `quimera` inalcanzables | nadie escribía `secreta`: dos plantillas eran datos muertos |

Las dos últimas se arreglaron **implementándolas**, no documentándolas: el fénix
pide tocar hambre 10 y volver a 0 en la misma sesión desde `salvaje` o `maratón`,
y la quimera dos temperamentos empatados al llegar a nivel 4. Las 27 plantillas
son ahora alcanzables.

### Lo que enseña

Los cuatro bugs de la primera ronda eran de **entrada externa mal validada**.
Doce de estos quince también. La diferencia es que en la primera ronda había una
sola entrada —el JSON de stdin— y en esta hay cuatro: stdin, `pet.json`, el
fichero de sesión y el JSON del hook. **Blindé la que ya conocía y no las tres
nuevas.** La lección no es "validar más": es que cada fichero que se añade es una
frontera de confianza nueva, y conviene contarlas.

---

## Los dos puntos de diseño, cerrados

### 5 · El k.o. ya es alcanzable

Exigir el 100% de la **media** era exigir los tres consumos al 100% a la vez: con
ctx, 5h y 7d al 100, 90 y 90 la media daba 95, o sea *ahogada*. El sprite en el
que más trabajo se invirtió no se veía nunca.

Ahora el k.o. tiene **puerta propia**: salta en cuanto el contexto llega al 100%,
sin mirar la media. Es coherente con por qué la media pondera 50/30/20 — el
contexto es lo único que te para de verdad — y no toca ningún otro estado.

> **Nota posterior.** Esa puerta ya no existe, y este apartado explica por qué
> hizo falta: era el síntoma, no la enfermedad. La causa era la media, que no
> puede llegar a 100 si no llegan los tres consumos. El uso volvió a ser el
> **cuello más apretado** —lo que medía la primera versión—, que llega a 100 él
> solo, así que la puerta sobraba y se fue con ella. Ver
> [design/vitals.md](design/vitals.md).
>
> **Y una tercera.** El cuello tampoco se quedó. Las cuotas de 5h y 7d son de la
> **cuenta**, no de la sesión, así que todas las ventanas abiertas leían el mismo
> número y la mascota dejaba de describir la suya. El uso es ahora el contexto de
> la sesión y nada más; el k.o. sigue sin necesitar puerta, porque el contexto
> llega al 100 él solo igual que el cuello.

### 6 · La calma es el defecto

La variable de calma pasó de ser un apaño opcional a no existir (se perdió al
mover el dibujo a su propio módulo) y luego a existir otra vez. Ahora está resuelto al
revés: **por defecto anda cuatro segundos de cada doce**, y la variable de andar
devuelve el baile continuo que pedía el diseño. Un movimiento perpetuo en la
esquina del ojo a 1 fps es un coste de atención permanente a cambio de nada.

## Y una lección que no es de código

Al hacer estos cambios descubrí que **otra sesión de Claude estaba editando este
mismo repo a la vez** (`claude-code-themes-84`, rediseñando la salida como un pie
con fondo y raya). Mis parches y los suyos se aplicaron sobre el mismo árbol de
trabajo sin colisionar por pura suerte: usé reemplazo de cadenas con anclas que
seguían existiendo.

Que funcionara no lo hace correcto. Lo que hay que hacer antes de editar un
fichero en un repo compartido es mirar `git status` **y** si hay otras sesiones
vivas, no descubrirlo a mitad de camino porque la salida no cuadraba.
