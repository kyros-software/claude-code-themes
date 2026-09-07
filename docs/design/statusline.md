# La statusline, banda por banda

Por qué cada dato está donde está, y qué se comprueba antes de pintarlo. El
[README](../../README.md) dice qué lleva cada banda; esto dice por qué.

```
──────────────────────────────────────────────────────────────────────────────────────────────────────────────
 Opus 5  ██████░░░░░░░░░░ 36% · 1M ctx │ xhigh │ 5h 41%  7d 13% │ 98% cache                           ▚╲   ╱▞
claude-code-themes (main) │ +184/−37 │ $28.29 │ 1h 12m                                                ▗▟███▙▖
criterio                                                                                             ▐█ > < █▌
cazabugs nivel 4 │ vibrante                                                                           ▖▖▀▀▀▗▗
```

Es un **pie**, no una línea más del hilo: fondo un tono por encima del negro y una
raya fina arriba. Cinco filas — la raya y cuatro bandas —, con la mascota anclada a
la derecha ocupando las cuatro. Cada banda agrupa datos que se miran juntos, y
suelta los elementos de menor prioridad antes que hacer *wrap*, que descuadra la
caja del prompt.

## Banda 1 · el motor

Las cuotas van como número pelado, sin barra, y **pintadas con la misma escalera**
que la barra de contexto y la mascota: un `5h` al 95% sale en el índigo de *ahogada*,
así que lo que está a punto de pararte es el color más fuerte de la línea aunque la
mascota esté verde. Es la respuesta a «¿por qué está ahogado si la ventana está
vacía?».

**El `tok/s` es real, no una estimación**, y hay que mirar de dónde sale. Los dos
campos del payload no miden lo mismo: `total_output_tokens` es lo que sacó la
*última* respuesta —se reinicia cada turno, no es un contador que sube— mientras
`total_api_duration_ms` es el tiempo de API *acumulado* de la sesión. El ritmo de
la última respuesta es lo primero entre lo que ha crecido lo segundo. Restar dos
`total_output_tokens` seguidos no mide nada: son dos respuestas distintas, y el
resultado sale inflado o negativo según cuál fuera más larga.

Se apaga solo a los dos minutos sin moverse, y entonces el acierto de caché ocupa
ese hueco. Nunca salen los dos.

La barra mide el mismo número que decide la mascota, así que **barra y mascota no
pueden contradecirse**. Las otras dos disposiciones se probaron y se leyeron como
un fallo: con la barra midiendo el contexto y tomando prestado el color del cuello,
una sesión al 48% con la cuota de 5h al 67 dibujaba una barra a media asta junto a
la palabra `espesa`, que es la lectura del 67. Ascendiendo la barra al cuello para
cerrar esa grieta, la banda imprimía `82% 5h` tres columnas antes de imprimir
`5h 82%` otra vez.

## Banda 2 · el trabajo

El nombre del repo lo da `workspace.repo.name` del payload cuando hay remoto, y si
no, la carpeta raíz. Solo el nombre: el owner es siempre el mismo y no te dice
dónde estás.

## Banda 3 · dónde y con qué criterio

La carpeta y el estilo de salida activo: lo que casi no se mueve.

De la ruta sale únicamente **la carpeta en la que estás**, y si se llama igual que
el repo —es decir, estás en su raíz— desaparece, porque eso ya lo dice la banda 2.

Los dos salen a pelo, sin etiqueta, y quien los distingue es el color: la carpeta
en gris porque es un sitio, el estilo en el morado de `Mode` porque es un ajuste de
la CLI. Se leen en orden *dónde → quién*, y cuando la banda se queda corta cae
antes el estilo: la banda era de la carpeta primero.

### Por qué el estilo va en minúscula

Es la voz del pie, no un dato del estilo. Todo lo demás que ocupa ese sitio ya
llega en minúscula —`xhigh`, `plan`, `auto-edit`, el `cazabugs` de la mascota—, así que
un nombre capitalizado sería la única palabra de la línea que grita.

Se hace en la banda y no al leer el payload por dos razones: `Payload.Style`
conserva el nombre real, y así entran también `Explanatory` y `Learning`, que
vienen capitalizados y **no se pueden renombrar**. La carpeta de al lado no se
toca: tiene que coincidir con lo que dice `ls`.

### Por qué el nombre se comprueba contra el disco

El payload manda el nombre **configurado**, no el cargado. En el CLI son dos pasos
y solo llega el primero:

```js
let d = Tn()?.outputStyle || "default"
return e[d] ?? null              // e = los estilos que cargaron
...
output_style: { name: Xe }       // Xe = la config, en crudo
```

O sea que una errata en `settings.json`, o un archivo borrado, se reportan igual
que un estilo que funciona **mientras el system prompt se queda vacío**. Pintar ese
nombre sería repetir la afirmación en vez de verificarla.

Así que la banda lo busca ella: los estilos de fábrica resuelven sin archivo, y el
resto tiene que aparecer en `~/.claude/output-styles/` o en `.claude/output-styles/`
del repo, con la regla de nombre del propio CLI — el `name:` del frontmatter, y si
no el nombre del archivo sin `.md`, comparado **con mayúsculas y todo**, porque al
otro lado es una clave de objeto. Si no aparece, no se pinta.

Los estilos que trae un plugin se buscan **en flojo**: vale cualquier copia
instalada bajo `plugins/cache/`, sin averiguar qué versión está viva — eso es el
cargador de plugins entero, una vez por segundo. Un falso positivo ahí solo
significa pintar un nombre que existe en algún sitio; esconder un estilo que
funciona sería peor. Cuesta 0,16 µs si es de fábrica, 5,9 µs si acierta en el
directorio de usuario y 28 µs en el barrido completo.

Lo que **no** detecta: un estilo que resuelve pero que no está cargado *en esta
sesión* porque la config cambió después de arrancarla. No hay rastro barato que
distinga eso — `/output-style` reescribe ese mismo ajuste y sí se aplica en
caliente, así que por fecha las dos situaciones son idénticas. Se arregla
reabriendo, y la banda no finge saberlo.

Sin estilo puesto el payload no manda un hueco: manda la palabra `"default"`
—`output_style: {name: outputStyle || "default"}`, leído del binario, no supuesto—
y pintarla gastaría columnas en decir que no hay nada.

### La banda puede quedar vacía

En la raíz de un repo y sin estilo, que es la mayoría de las sesiones. Esa fila se
ancla con un **braille en blanco** (`U+2800`), porque Claude Code recorta los
espacios de la izquierda y sin él el trozo de mascota de esa fila se cae al borde.

## Banda 4 · la mascota

```
cazabugs[sabueso] nivel 5 │ fresca ✦ │ ████░░░░ │ ◗ cinco días de racha
```

El corchete escribe **la marca que la mascota lleva puesta**, con el oficio del que
es variante fuera: se lee entero como un nombre, *un cazabugs, en su forma
sabueso*. El árbol se bifurca en los niveles 2, 3 y 5, y la marca es la del 5, así
que el corchete sale ahí y en ningún otro sitio: `cazabugs` en el nivel 4,
`cazabugs[sabueso]` en el 5, y `lobo` a secas en el 6, donde el título es el final
de la rama y no necesita contexto.

**Decía lo contrario.** Escribía la marca a la que la mascota *apuntaba*, de modo que
un nivel 4 leía `cazabugs[sabueso]` sin ser un sabueso. La idea era que el corchete
fuese el tiempo verbal —un nombre dice *es*, un corchete dice *va hacia*—, y eso
solo funciona si se ve: iba pintado en el color de la barra separadora, **1,54:1**
contra el fondo frente al **11,8:1** de las dos palabras que lo rodean. Dos
palabras brillantes pegadas sin nada visible en medio se leen como un nombre
compuesto, que es exactamente lo que era.

**El estado vive aquí, no coronando a la mascota.** El lienzo lo dibuja dos veces, pero
en una terminal de verdad la misma palabra acaba en el mismo pie a pocas columnas
de sí misma y se lee como un fallo. Bajarlo a la banda le devolvió a la mascota la fila
que necesita la cresta.

La barra mide **el tramo de este nivel**, no la xp total, así que amanece vacía el
día después de subir. En el tope, donde ya no queda escalera, cambia de moneda:
pasa a medir el **hábito** que abre la siguiente marca, en ámbar y con su nombre al
lado. Una mascota que ya lleva la suya no tiene ninguna de las dos, y entonces la
banda se sostiene sobre el estado.

## Anchos

| Columnas | Qué pasa |
| --- | --- |
| < 100 (`BubbleMin`) | la banda 4 se queda **solo con el oficio** |
| < 55 (`minWidthForPet`) | la mascota desaparece y quedan las cuatro bandas |

**El margen derecho.** La statusline no recibe el ancho de la terminal —no hay
campo para eso en el JSON—, así que sale de `COLUMNS`. Y Claude Code recorta la
línea unas 5 columnas antes, de modo que alinear sobre `COLUMNS-1` trunca la mascota
o lo hace *wrap*. De ahí el margen de 6 por defecto (`STATUSLINE_RIGHT_PAD`).

**Los espacios de la izquierda.** Claude Code los recorta. Las filas cuya mitad
izquierda va vacía son solo "espacios + mascota": al recortarlos, la mascota se cae al
borde y acabas con trozos sueltos por la pantalla. Por eso el braille en blanco.

## Lo que no está en su mano

- La línea de `bypass permissions` y los badges tipo `/rc active` los pinta Claude
  Code en su propio footer. Por eso el modo de permisos sale como **marca** y no
  como palabra: en bypass, un `⚡` rojo, en vez de deletrear otra vez lo que ya está
  escrito tres líneas más arriba. `plan` y `auto-edit` conservan su nombre: no
  tienen glifo evidente y uno inventado sería un acertijo.
- **El techo de la animación son 1 fps.** Se re-ejecuta por eventos (con *debounce*
  de 300 ms) y en reposo solo si defines `refreshInterval`, cuyo mínimo es 1 s.
- El **banner de bienvenida** usa acentos de onboarding que no forman parte del
  sistema de temas: se queda en el rosa de marca con cualquier tema activo.

El modo de permisos no viene en el payload, pero sí en el transcript, cuya ruta sí
llega. Se lee solo la cola del fichero (0,02 ms).
