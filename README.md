# claude-code-themes

Tres temas de color para [Claude Code](https://claude.com/claude-code) y una
**statusline** que ocupa el pie de la ventana: cuatro bandas de datos a la
izquierda y una mascota a la derecha que refleja el estado de la sesión y
**evoluciona según cómo trabajas** — 41 formas en un árbol de seis niveles, con
xp, hambre y racha.

Plugin instalable. El runtime es un binario de Go sin dependencias: ni `python3`,
ni `node`, ni `jq`. Cuesta 1,5 ms por refresco.

```
──────────────────────────────────────────────────────────────────────────────────────────────────────────────
 Opus 5  ██████░░░░░░░░░░ 36% · 1M ctx │ xhigh │ 5h 41%  7d 13% │ 98% cache                           ▚╲   ╱▞
claude-code-themes (main) │ +184/−37 │ $28.29 │ 1h 12m                                                ▗▟███▙▖
criterio                                                                                             ▐█ > < █▌
cazabugs nivel 4 │ vibrante                                                                           ▖▖▀▀▀▗▗
```

## Instalación

```
/plugin marketplace add gabriel-diagram/claude-code-themes
/plugin install claude-code-themes
/pet-statusline
```

Lo primero trae los temas, los comandos y los hooks que alimentan a la mascota. Lo
tercero enciende la statusline — hace falta aparte porque `statusLine` no es un
componente de plugin y la clave va en `~/.claude/settings.json`, con copia de
seguridad y escritura atómica. Después, `/theme` → Terminal.

Sin plugin:

```bash
scripts/install.sh            # temas + statusline + mascota + /pet y /feed
scripts/install.sh --hooks    # además engancha los hooks que le dan de comer
scripts/install.sh --uninstall
```

Los **hooks van aparte a propósito**: viven en el `settings.json` global, así que
corren en todos tus repos. Sin ellos la mascota existe y se ve, pero solo come con
`/feed`.

## Las cuatro bandas

- **1 · el motor** — modelo, contexto, los dos límites, razonamiento y ritmo: lo
  que cambia cada turno. Las cuotas van pintadas con la misma escalera de color que
  la mascota, así que un `5h` al 95% sale en índigo aunque la ventana esté vacía.
- **2 · el trabajo** — repo, rama, diff, coste y el reloj de la sesión.
- **3 · dónde y con qué criterio** — la carpeta (solo la carpeta; si estás en la
  raíz del repo, desaparece) y el estilo de salida activo, **comprobado contra el
  disco** antes de pintarlo: el payload manda el nombre configurado, no el cargado.
- **4 · la mascota** — oficio, la marca entre corchetes, nivel, cómo está, la barra y
  el bocadillo. `cazabugs[sabueso]` se lee entero como un nombre: *un cazabugs, en
  su forma sabueso*.

Por debajo de **100 columnas** la banda 4 se queda solo con el oficio; por debajo de
**55**, la mascota desaparece.

El porqué de cada decisión está en [statusline.md](docs/design/statusline.md).

## La mascota

Nueve columnas. La silueta **y el color** los elige la evolución; los ojos, las
patas y el peldaño de la rampa los elige el estado. Cada rama tiene su tono y lo
mantiene en los siete estados, que es lo que permite distinguir 41 siluetas en las
filas que hay.

### Siete estados

Un solo número decide estado, ojos, patas y color: `context_window.used_percentage`.
Cambia **cuatro señales independientes**, en este orden — primero los ojos, luego el
paso, luego la cabeza se hunde, y al final la silueta se tumba. A un vistazo se
distingue *cansada* de *ahogada* sin leer la etiqueta.

| Uso | Etiqueta | Ojos | Cabeza | Patas |
| --- | --- | --- | --- | --- |
| ≤22% | fresca ✦ | `>` `<` | sí | anda |
| ≤45% | vibrante | `>` `<` | sí | anda |
| ≤63% | a gusto | `o` `o` | sí | anda |
| ≤78% | espesa | `▬` `▬` | sí | quieto |
| ≤89% | cansada | `_` `_` | **hundida** | quieto |
| <100% | ahogada | `x` `x` | hundida | quieto |
| 100% | k.o. | `x` `x` | hundida | **tumbado** |

Es **solo el contexto**, a propósito. Las cuotas de 5h y 7d son de la cuenta, no de
la sesión: metiéndolas en la cuenta, todas las ventanas abiertas leían el mismo
número y la mascota dejaba de hablar de la sesión en la que vive. Siguen en la banda
1, con número y color, pero sin cara. El razonamiento entero en
[vitals.md](docs/design/vitals.md).

### El árbol

La forma no la eliges: sale de cómo trabajas. Los commits y los `/compact` llevan
por la rama **metódica**, los tests y los planes por la **inquisitiva**, y trabajar
con el contexto arriba por la **impulsiva**.

```
        nivel 1        2            3          5              6
        chispa ─┬─ brasa ─┬─ maratón ──┬─ buey ────── mamut
                │         │            └─ topo ────── gusano
                │         ├─ salvaje ──┬─ gremlin ─── diablo
                │         │            └─ kraken ──── leviatán
                │         └─ velocista ┬─ francotirador ─ halcón
                │                      └─ relámpago ─ tormenta
                ├─ pauta ─┬─ pulcro ───┬─ jardinero ─ bosque
                │         │            └─ monje ───── abad
                │         └─ refactor ─┬─ cirujano ── bisturí
                │                      └─ tejedor ─── telar
                └─ sonda ─┬─ arquitecto┬─ cartógrafo ─ atlas
                          │            └─ oráculo ─── esfinge
                          └─ cazabugs ─┬─ exterminador ─ avispa
                                       └─ sabueso ──── lobo
```

1 raíz + 3 temperamentos + 7 oficios + 14 marcas + 14 títulos = **39**, más dos
secretas fuera del árbol: `fénix` y `quimera`. El nivel 4 no bifurca.

Cada fila del lienzo es una forma en los siete estados. **La marca de arriba y el
número de patas identifican la forma y no cambian nunca**; el estado rellena los
ojos, mueve el paso, aplana el cuerpo a partir de *cansada*, lo tumba en *k.o.* sin
perder la cuenta de patas, y baja el color por la rampa de esa rama.

![Nivel 1: chispa](assets/formas-nivel-1.png)
![Nivel 2: los tres temperamentos](assets/formas-nivel-2.png)
![Nivel 3: los siete oficios](assets/formas-nivel-3a.png)
![Nivel 3: los siete oficios, continuación](assets/formas-nivel-3b.png)

Las marcas y los títulos heredan la rampa de su oficio —un sabueso es azul como el
cazabugs del que sale, y lo que los distingue es el cuerpo—, y por eso **diez rampas
bastan para 41 formas**. Todas salen de `internal/pet/testdata/ATLAS.json`, que está
en el repo y contra el que cuatro tests comparan el Go: los 41 nombres, las 10
rampas, los padres del árbol y las 287 siluetas fila a fila.

### Cómo come

| Evento | xp | Hambre | Freno |
| --- | --- | --- | --- |
| tests en verde | **+15** | −4 | una vez por hora, y solo si has cambiado algo |
| commit | **+12** | −3 | — |
| compact | **+8** | −3 | — |
| tarea del plan cerrada | **+6** | −1 | — |
| `/feed` | **+3** | −2 | uno cada cuatro horas |
| contexto al 100% | **−15** | — | rompe la racha |

Los niveles llegan a 60, 180, 400, 2000 y 4500 xp.

**Y baja.** El hambre sube +1 por hora sin comer, con tope en 10; a partir de ahí
cada hora cuesta 1 xp. La xp tiene techo —el último umbral más un tramo de nivel 1,
`4500 + 60`—, porque sin él el colchón acumulado se traga cualquier castigo. De ahí
las dos cifras: 60 horas —**dos días y medio**— para perder el nivel de arriba, y
4560 horas, **unos seis meses**, para volver a larva. Nunca muere: por abajo se
queda en `chispa`, que es una forma, no una tumba.

**Una forma no se cae.** Se mueve en lateral, hacia arriba o de rama: un
`exterminador` pasa a `sabueso` o a `avispa`, pero nunca vuelve a `cazabugs`
pelado. Baja un peldaño en un solo caso —cambias de rama y allí ya tienes una
marca ganada—, porque eso no es caerse, es haberte movido. El nivel sí puede
bajar aunque la forma no, así que `avispa nivel 5` es legítimo.

El árbol entero y qué alimenta cada contador, en
[evolution.md](docs/design/evolution.md).

### `/pet`

Enseña los **22 contadores que deciden el árbol**, no solo los cuatro de la
statusline. El color significa una sola cosa: si ese contador te lleva a algún sitio
al que todavía puedes llegar.

```
  cazabugs   nivel 4
  inquisitivo › sonda › cazabugs

  nivel  █░░░░░░░░░░░░░░░  533/2000 xp
  hambre ██░░░░░░░░  2
  racha  ███░░░░  3 días · mejor 3

  la marca del nivel 5
    ✓ sabueso        reproducir antes de arreglar  36/10
      exterminador   días seguidos en verde         2/15
```

## Los temas

| Tema | Acento | Look |
| --- | --- | --- |
| **Terminal** | `#4dd6c1` turquesa | un color por tipo de dato |
| **Blood Red** | `#ff5c47` coral | cálido: coral, terracota, vino |
| **Electric Blue** | `#2e8bff` azul | frío: cian, azure, azul profundo |

![Electric Blue a la izquierda y Blood Red a la derecha](assets/preview.png)

*La statusline que asoma en la esquina de esa captura es vieja; los colores del CLI,
que es lo que enseña, siguen siendo estos.*

**Terminal** es el que empareja con la statusline: un tipo de dato lleva siempre el
mismo color, así no hay que leer para saber qué estás mirando.

| Rol | Hex | |
| --- | --- | --- |
| Rutas, ficheros, repos | `#4DD6C1` | turquesa |
| Identificadores, código, altas | `#57E389` | verde |
| Urls, ramas, enlaces | `#6FB6FF` | azul claro |
| Números, dinero, avisos | `#E8C46A` | ámbar |
| Modos y ajustes del CLI | `#B07CF0` | violeta |
| Bajas, errores, riesgo | `#F2777A` | salmón |
| Énfasis en prosa | `#ECEFF4` | casi blanco |
| Separadores, unidades | `#6B7683` | gris |

Los tres cubren los **72 tokens** que reconoce Claude Code, no solo la docena que se
ve de un vistazo.

## Ajustes

| Variable | Efecto |
| --- | --- |
| `STATUSLINE_PET=0` | apaga la mascota, deja las cuatro bandas |
| `STATUSLINE_PET_WALK=1` | anda en cada refresco en vez de a ratos |
| `STATUSLINE_BACKGROUND=0` | quita el fondo del pie |
| `STATUSLINE_RULE=0` | quita la raya de arriba y ahorra una fila |
| `STATUSLINE_RIGHT_PAD` | margen derecho, por defecto `6` |
| `PET_TEST_RUNNERS` | regex extra para reconocer tu runner de tests |
| `CLAUDE_CONFIG_DIR` | mueve `~/.claude`; la mascota y la statusline lo respetan |

**Truecolor.** Los temas usan color de 24 bits, y Windows Terminal, WSL y `docker
run` no exportan `COLORTERM`. Sin él los tonos parecidos colapsan al mismo:

```bash
export COLORTERM=truecolor                                  # .zshrc / .bashrc
docker run -e COLORTERM=truecolor -e TERM=xterm-256color ...
```

La mascota sí tiene plan B: cuantiza al cubo de 256 de verdad, así que se ve igual,
con menos tonos.

## Migración desde la versión en Python

**No hay que hacer nada.** `scripts/install.sh` borra los lanzadores viejos y el
`pet.json` se traduce solo la primera vez que se escribe. La mascota conserva xp,
racha, contadores y forma secreta. Lo que se lee en pantalla está en castellano; el
fichero guarda los ids en inglés, porque renombrarlos reescribiría todos los
ficheros de vida que hay por ahí.

Lo único a mano son las variables de entorno: las que estaban en español ya no
se leen, y ahora son `STATUSLINE_PET`, `STATUSLINE_PET_WALK`,
`STATUSLINE_BACKGROUND` y `STATUSLINE_RULE`.

## Más a fondo

- [statusline.md](docs/design/statusline.md) — las cuatro bandas: por qué cada dato
  está donde está y qué se verifica antes de pintarlo
- [vitals.md](docs/design/vitals.md) — la capa del momento: de fresca a k.o.
- [evolution.md](docs/design/evolution.md) — la capa permanente: xp, comida y las 41
  formas
- [runtime.md](docs/design/runtime.md) — por qué Go, a dónde va el tiempo, el
  candado del `pet.json` y por qué los binarios van en el repo
- [audit-log.md](docs/audit-log.md) — histórico: la auditoría de la versión Python
- [umbrales.md](docs/design/umbrales.md) — **sin implementar**: el árbol de 97 formas
  del lienzo de diseño, por qué su regla deja 27 de las 42 marcas fuera de alcance, y
  el arreglo que las devuelve. Las puertas viven en `testdata/PUERTAS-97.json` y las
  comprueba `go test ./internal/pet/ -run NinetySeven`

## Licencia

[MIT](LICENSE).
