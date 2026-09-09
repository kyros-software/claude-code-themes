# Las evoluciones de la mascota

La mascota tiene **dos capas que no se mezclan**:

| | qué mide | de dónde sale | sube y baja |
| --- | --- | --- | --- |
| **vida** | cómo está *ahora* | uso de contexto y cuota | sí, todo el rato |
| **progreso** | lo que llevas hecho | XP acumulada en `~/.claude/pet.json` | sube comiendo, baja de hambre |

La **vida** elige los ojos, las patas y el color — está en [vitals.md](vitals.md).
La **progreso** elige la silueta, que es de lo que va este documento.

## Una plantilla, siete estados

Cada evolución es una **plantilla de 5 filas y 9 columnas** con dos huecos de
ojos y una fila de patas:

```
  |   |     <- marca de la ramificación
 ▗█┼█┼█▖    <- cuerpo de la evolución
▐█ o o █▌   <- los ojos los pone el estado
 ▝█┼█┼█▘
 ▘▘   ▝▝    <- las patas las pone el estado
```

El estado de vida **no cambia la silueta ni el tono**: rellena los huecos y baja
por la rampa de la rama. El color pertenece a la evolución, no al estado. Por eso
las 41 evoluciones tienen sus siete estados sin dibujar 287 sprites.

Los ojos siguen una regla: la evolución pone los suyos mientras está entera
(*fresh* y *lively*), y de *easy* para abajo manda el estado
(`o o` → `▬ ▬` → `_ _` → `x x`). Así el cansancio se lee de un vistazo aunque
no sepas qué mascota es.

## El árbol

```
spark
├─ pattern  compacta y commitea corto
│  ├─ refactor  muchos diffs pequeños
│  │  ├─ surgeon.......  20 diffs seguidos sin un rechazo
│  │  └─ weaver........  un refactor que toca 10+ ficheros
│  └─ tidy  jamás pasa del 60%
│     ├─ monk..........  5 sesiones sin pasar del 40% de contexto
│     └─ gardener......  docs y limpieza dos días seguidos
├─ probe  lee, planifica, testea
│  ├─ bughunter  tests y fixes en cadena
│  │  ├─ bloodhound....  repro antes del fix, 10 veces
│  │  └─ exterminator..  15 tests verdes sin uno rojo
│  └─ architect  planes largos, docs
│     ├─ cartographer..  un plan de 10 tareas cerrado entero
│     └─ oracle........  5 planes escritos antes de tocar código
└─ ember  tira al límite sin frenar
   ├─ sprinter  sesiones cortas, rápidas
   │  ├─ bolt..........  10 sesiones de menos de 15 minutos
   │  └─ sniper........  8 tareas cerradas con una sola herramienta
   ├─ marathon  sesiones de horas
   │  ├─ ox............  3 sesiones de más de 4 horas
   │  └─ mole..........  5 días seguidos en el mismo repo
   └─ feral  al límite, sin compactar
      ├─ gremlin.......  30 turnos con permisos en bypass
      └─ kraken........  3 sesiones tocando el 100% de contexto
```

### Los títulos

Detrás de cada marca hay un título, y pide **más del mismo hábito**. Los catorce
números salen del lienzo [«Cómo llegar a cada forma»][canvas], que da un factor
por título en vez de un multiplicador único: cada hábito se pesó por separado.

| marca | título | pide | | marca | título | pide |
| --- | --- | --- | --- | --- | --- | --- |
| `surgeon` 20 | `scalpel` | **50** diffs seguidos | | `cartographer` 10 | `atlas` | **20** tareas en un plan |
| `weaver` 10 | `loom` | **25** ficheros de un tirón | | `oracle` 5 | `sphinx` | **20** planes antes de código |
| `monk` 5 | `abbot` | **15** sesiones bajo el 40% | | `bolt` 10 | `storm` | **30** sesiones de menos de 15 min |
| `gardener` 2 | `forest` | **7** días de docs | | `sniper` 8 | `falcon` | **25** tareas de una herramienta |
| `bloodhound` 10 | `wolf` | **30** repros antes del fix | | `ox` 3 | `mammoth` | **10** sesiones de más de 4 h |
| `exterminator` 15 | `wasp` | **50** tests verdes seguidos | | `mole` 5 | `worm` | **20** días en el mismo repo |
| | | | | `gremlin` 30 | `devil` | **200** turnos en bypass |
| | | | | `kraken` 3 | `leviathan` | **10** sesiones al 100% |

Dos no son el número del lienzo, y conviene saber por qué:

- **`atlas`.** El lienzo pide «5 planes de 10 tareas cerrados», que es una
  *cuenta* de planes grandes. El contador que hay, `longest_plan`, es el plan
  más largo cerrado nunca — un máximo, no una cuenta. Poner 50 ahí pediría un
  solo plan de cincuenta tareas, que es otra cosa, no una más difícil.
- **`devil`.** El lienzo pide 100 turnos en bypass. `bypass_turns` sube unas
  treinta veces al día — medido — así que 100 es un título que llega ya
  cumplido, y eso no es un título. 200 lo pone en una semana, como los demás.

[canvas]: https://claude.ai/design/p/4639e060-9aec-4ae3-855a-f8530ae9ab34

**La rama no la eliges.** En cada bifurcación gana el contador de comportamiento
que va más alto en ese momento. Si cambias de hábitos antes de subir de nivel,
cambias de rama.

### «Más alto» no es el número crudo

Los contadores que leen las bifurcaciones **no son la misma clase de número**.
Cuatro suben una vez por *evento* y no paran —un commit, un `/compact`, una
tarea de plan cerrada— y cinco suben como mucho una vez por *sesión*, al
cerrarla. Un día tiene una docena de commits y tres o cuatro sesiones, así que
comparar `methodical` con `impulsive` en crudo lo decidían las unidades antes de
que el hábito abriera la boca.

Medido en un `pet.json` real, después de tres semanas trabajando exactamente al
límite: `methodical` 39, `impulsive` 2. Y el 2 no es falta de ganas, es el techo.
Para ser `brasa` había que acumular más sesiones al límite que commits+compacts
**en toda la vida de la mascota**, lo cual se llevaba por delante 3 oficios, 6
marcas y 6 títulos: **16 de las 41 formas**, inalcanzables jugando.

Es el mismo fallo que se arregló en el nivel 5 —«gana el primero que cruza su
umbral» — un peldaño más arriba y peor, porque aquí no hay umbral que cruzar y
por tanto nada contra lo que normalizar. Ahora cada contador se divide por su
**escala**, que es lo que da un día de ese hábito, y la bifurcación vuelve a ser
una carrera entre formas de trabajar:

| Contador | Escala | De dónde sale |
| --- | --- | --- |
| `methodical` | 10 | commits a 12 xp, más los compacts |
| `inquisitive` | 8 | suites verdes a 15 xp — donde cae también el freno horario |
| `diffs` | 10 | solo commits |
| `tests` | 8 | el mismo freno |
| `plans` | 21 | tareas de plan a 6 xp |
| `impulsive` | 3 | una por sesión ≥85% |
| `ctx_low` | 3 | una por sesión <60% |
| `ctx_maxed` | 3 | una por sesión ≥95% |
| `short_sessions` | 3 | una por sesión <15 min |
| `long_sessions` | 2 | una por sesión ≥90 min — caben menos en un día |

Las cinco primeras salen del presupuesto que el diseño ya tenía —un día normal
son 128 XP, el mismo número detrás del freno horario de la suite— dividido por lo
que paga cada comida. Las cinco de sesión no pueden pasar de cuántas sesiones
caben en un día, medido en tres o cuatro. Son una **calibración, no una ley**, y
viven en `BranchScale` (`internal/pet/evolution.go`).

### Y una rama tomada se defiende

Dividir por la escala arregla *quién* gana la bifurcación, no *cada cuánto*
cambia de manos. Dos hábitos que van a la par no se mantienen delante el uno del
otro mucho rato: medido, `methodical` 4,60 días contra `inquisitive` 4,38, que
son 0,22 de diferencia — **dos suites en un sentido, tres commits en el otro**.
La mascota cambiaba de nombre y de sprite varias veces en una tarde, en una
bifurcación que en ningún momento se había decidido de verdad.

Así que una rama tomada **se defiende**: para quitársela hay que sacarle un día
entero del hábito (`BranchMargin`, 1.0, en la unidad en la que ya están las
escalas). Un día es la elección y no medio, porque medio día son cinco commits y
eso cabe en una tarde; un día de hábito no se cruza ida y vuelta dentro de una
sesión. Comprobado sobre el `pet.json` real:

| ventaja de inquisitivo | forma |
| --- | --- |
| +0,80 días | `pulcro` |
| +0,92 días | `pulcro` |
| **+1,05 días** | `cazabugs` |

Es un retardo, no un candado: los contadores solo suben, así que cualquier rama
se acaba pudiendo tomar — `TestADefendedForkCanStillBeTaken` lo exige de los dos
lados de cada bifurcación, contra cada defensor posible. Lo que cambia es el
plazo: el jugador de `brasa` del test cruza el nivel 2 el día 2 llevando la rama
que iba un pelo por delante en ese instante, y no le quita la bifurcación hasta
el día 8, cuando la ventaja es un día entero.

**El precio, con los ojos abiertos.** Los contadores no pueden decir quién iba
delante ayer —solo suben—, así que la decisión hay que **guardarla**: `branch` en
`pet.json`, un hijo por bifurcación cruzada. Eso rompe una propiedad que el resto
de este fichero sí cumple: la forma deja de ser función pura de los contadores, y
**dos mascotas con los mismos números pueden llevar formas distintas** según por
dónde pasaron. Es el único dato de una mascota que está solo en el fichero. Se
sabía antes de elegirlo.

Lo escribe `RememberBranch`, y lo escribe `Save` —no cada llamante—, por la misma
razón que `RememberForm`: seis caminos persisten este fichero y solo dos tenían
motivo para pensar en ramas. Una bifurcación que no se anota es una bifurcación
sin defensor, o sea la histéresis no ocurriendo en silencio. Y anota solo las
bifurcaciones **ya cruzadas**: apuntar la del nivel 3 con la mascota en el 2 le
daría un defensor elegido un nivel antes de tiempo.

En la entrada, el campo pasa por una puerta más estricta que la del peldaño: la
pareja tiene que nombrar una elección real —una bifurcación que el árbol tenga, y
un hijo suyo— o se tira. Las bifurcaciones del nivel 5 no entran: esas las decide
`ripestMark` y no se defienden.

Lo que las defiende es un test que juega, no uno que rellena contadores a mano:
`TestTheEmberBranchSurvivesANormalDayOfWork` simula a alguien que trabaja al
límite **y además commitea**, y exige las dos direcciones —que llegue a `brasa`
con un par de commits al día, y que **no** llegue si commitea todo el día—.
Porque dividir no es poner el pulgar en la balanza.

**Por qué los tests viejos no lo veían.** `TestEveryFormIsReachableFromAVeteran`
escribe el contador a mano (`steer`, «un punto por encima del hermano»), así que
prueba «alcanzable si el número fuera más alto», nunca «el número puede llegar
ahí jugando». Y `TestEveryTemperamentIsReachableByPlaying` sí jugaba, pero solo
al jugador **puro**: su caso de `ember` no cierra nada en absoluto, solo `feed` y
pico alto. La rama parecía viva y no la podía tomar nadie que trabajase.

**El empate de la quimera también se escaló.** Comparaba los tres temperamentos
en crudo, así que la secreta le tocaba a quien tuviera por casualidad el mismo
número de commits que de suites — una coincidencia de unidades, no dos formas de
trabajar que salieron a la par. Ahora empatan en días. Las quimeras ya
concedidas se quedan: `CheckSecrets` no reescribe una secreta ya puesta.

**Y las tres ramas se pueden tomar.** La impulsiva estuvo muerta: su contador
solo lo subía *reventar el contexto*, que es la única comida que **resta** XP.
La aritmética se cerraba sola —cada punto de `impulsive` costaba 15 XP, y toda
comida que los devolvía alimentaba a un rival—, así que quien reventaba el
contexto todo el día acababa con 800 de impulsivo y clavado en el nivel 1 con
0 XP. Un tercio del árbol detrás de una rama que nadie podía subir.

Ahora paga el **pico de contexto de la sesión**: a partir del `ImpulsivePeak`
(85%) cuenta como haber trabajado al límite, que es lo que pide el lienzo
—«tira al límite sin frenar»— y no lo mismo que estrellarse. Es el espejo de
`ctx_low`, que premia al que no pasa del 60 y lleva a `pulcro`.

**Y un nivel más abajo pasaba lo mismo.** `ctx_maxed`, que elige `feral` entre
sus dos hermanos, tenía la aritmética cerrada exactamente igual: su única fuente
era el reventón, mientras que `short_sessions` y `long_sessions` se cobraban
gratis con solo tener sesiones. Ahora los tres contadores que leen la rama brasa
son **tres muescas del mismo gesto**, y ninguno se paga en XP:

| Pico de la sesión | Contador | Para qué |
| --- | --- | --- |
| ≥ 85 (`ImpulsivePeak`) | `impulsive` | nivel 2 — `ember` |
| ≥ 95 (`FeralPeak`) | `ctx_maxed` | nivel 3 — `feral` |
| ≥ 100 | `ctx100_sessions` | la marca `kraken` |

El reventón (`overflow`) ya no alimenta ningún hábito: es lo que siempre debió
ser, un castigo de −15 XP que además rompe las rachas limpias.

**Cuidado con el empate.** Una sesión de más de 90 minutos con el pico arriba
sube `ctx_maxed` *y* `long_sessions`, uno cada uno, y los contadores quedan
empatados para siempre — pero gana `marathon`, porque en un día caben menos
sesiones largas que sesiones, y una sesión larga es por tanto más día de trabajo
que una al límite. `feral` es para quien llena la ventana **rápido**: sesiones
cortas al límite. Es la distinción que la rama está dibujando, y hay un test que
la fija (`TestALongSessionAtTheLimitStillGoesToMarathon`).

**Los nombres del árbol son ids, no texto.** `spark`, `bughunter` o `exterminator`
son lo que hay escrito en `pet.json` desde la versión en Python, y renombrarlos
reescribiría todos los ficheros de vida que hay por ahí. Lo que se lee en
pantalla es la columna en castellano del lienzo —*chispa*, *cazabugs*,
*exterminador*—, y vive en `internal/pet/names.go`. Una forma sin traducir cae
en su propio id: un nombre que falta es un nombre por elegir, no un fallo.

Cuál de las dos marcas te toca no lo decide la XP sino el **hábito**: se
desbloquean al cumplir su condición estando en la evolución padre. La XP sigue
poniendo el *cuándo* —son formas de nivel 5— pero ya no reparte nada: a partir
de ahí lo único que se mueve es el hábito, y es eso lo que mide la barra en la
banda 4 y la fila `marca` de `/pet`.

| Nivel | XP | Tramo | Qué eres |
| --- | --- | --- | --- |
| 1 | 0 | — | larva — `spark`, sin patas todavía |
| 2 | 60 | 60 | temperamento — cómo trabajas |
| 3 | 180 | 120 | oficio — en qué eres bueno |
| 4 | 400 | 220 | el mismo oficio, asentado |
| 5 | 2000 | **1600** | marca — más la condición de hábito, o una secreta ya ganada |
| 6 | 4500 | 2500 | título — la forma final de la rama |

### Por qué el nivel 4 es tan largo

El árbol **solo se bifurca en los niveles 2, 3 y 5**. El 4 no reparte nada: es
el mismo oficio, asentado. Eso lo convierte en el único tramo donde el hábito
que decide la marca sigue moviéndose y **todavía puede cambiar de idea** — un
`cazabugs` que empieza a reproducir fallos antes de arreglarlos se escora a
`sabueso`, uno que encadena suites verdes se escora a `exterminador`, y
cualquiera de los dos puede adelantar al otro mientras dure el nivel.

Duraba 500 XP. Medido contra un día real de comidas —unos 12 XP por comida, del
orden de 60 a la hora de trabajo efectivo— eso son **ocho horas**: una sesión
larga, y la bifurcación decidida antes de que los hábitos hubieran tenido una
semana para decir nada. Con 1600 el tramo ronda las **veinticinco horas** de
trabajo.

Durante ese tramo la banda 4 dice `cazabugs` y nada más: la barra de XP es la
que enseña el avance, y cuál de las dos marcas se está ganando no se anuncia
—sigue en disputa hasta el final, que es justo lo que hace largo al nivel 4—.
La fila `marca` de `/pet` sí lleva la cuenta de los dos hábitos.

### La variante que llevas: `cazabugs[sabueso]`

La banda 4 escribe entre corchetes **la marca que la mascota lleva puesta**, con
el oficio del que es variante fuera:

```
cazabugs[sabueso] nivel 5 │ fresca ✦
```

Se lee entero como un nombre: *un cazabugs, en su forma sabueso*. El árbol se
bifurca en los niveles 2, 3 y 5, y la marca es la bifurcación del 5 — así que
el corchete aparece ahí y en ningún otro sitio:

| Nivel | Banda 4 | Por qué |
| --- | --- | --- |
| 4 | `cazabugs` | todavía no hay variante que nombrar |
| 5 | `cazabugs[sabueso]` | la bifurcación, y por cuál fue |
| 6 | `lobo` | el título es el final de la rama y no compite con nada |

**El corchete decía lo contrario y engañaba.** Escribía la marca a la que la
mascota *apuntaba*, así que un nivel 4 leía `cazabugs[sabueso]` sin ser un sabueso
y sin garantía de llegar a serlo. La intención era que los corchetes fuesen el
tiempo verbal —un nombre dice *es*, un corchete dice *va hacia*— pero eso solo
funciona si se ven: iban pintados en `Rule`, el color de la barra separadora,
que da **1,54:1** contra el fondo frente al **11,8:1** de las dos palabras entre
las que se sientan. Lo que llegaba al ojo eran dos palabras brillantes pegadas
sin nada en medio, y se leía como un nombre compuesto. La puntuación que
cargaba todo el significado era lo único invisible. Ahora van en `Dim`.

Es lo primero que la banda suelta al quedarse sin columnas, después del
bocadillo. Perder el corchete cuesta algo real —cuál de las dos formas tomó el
oficio— pero la mitad que queda en pie sigue siendo **cierta**: un sabueso es un
cazabugs, así que `cazabugs` a secas es menos preciso y no es falso.

Y dos secretas fuera del árbol: **phoenix** / *fénix* (llegar a hambre 10 y
remontar a 0 en la misma sesión, solo desde `feral` o `marathon`) y **chimera** /
*quimera* (dos temperamentos empatados al subir a nivel 4; hereda ojos de uno y
cuerpo del otro).

Las dos son formas de **nivel 5** y esperan a los 2000 XP como cualquier otra. La
condición se cumple antes —la de la quimera, a nivel 4— y entre una cosa y la
otra el panel dice a qué aspiras: `488 para quimera`. Entregarla en el acto se
saltaba el nivel 4 entero y ponía un «nivel 5» al lado de 412 XP.

### Una secreta gana su peldaño, no el de arriba

Y era el final del camino. `walk` devolvía la secreta **antes** de recorrer el
árbol, así que la mascota se quedaba en el peldaño 5 para siempre: ni marca, ni
título, y la rama en la que estaba dejaba de significar nada el día que le tocó
la secreta. Tres cuartos de rama a cambio de una forma bonita. Una quimera es una
forma de nivel 5, no una lápida.

Ahora el árbol se recorre primero y la secreta solo se pone por encima si el
árbol devolvió peldaño 5 **o menos**. El título es peldaño 6 y la supera, así que
sigue siendo algo en lo que una quimera puede convertirse:

| Peldaño que da el árbol | Lo que lleva puesto | Por qué |
| --- | --- | --- |
| oficio (3) o marca (5) | la secreta | es más rara, y es su peldaño |
| título (6) | el título | está por encima, y se paga con el hábito |

Las marcas que se salta por el camino no son una pérdida: la secreta ya ocupa ese
peldaño, y el título de detrás pide **el mismo hábito**, más cantidad. El hábito
sigue siendo la puerta; lo que cambia es la forma que llevas mientras la cruzas.
El panel lo dice —`22/50 para avispa` debajo de `fénix`—, que antes estaba
correctamente vacío porque no había nada a lo que apuntar.

Y una vez puesto, el título no se devuelve: `tradeOf` de una secreta es `""`,
que el suelo lee como «no sé de qué rama viene» y por tanto se queda con lo más
alto que pisó. Una racha que se cae no te baja de avispa a fénix.

## Una forma no baja de escalón

La forma se recalcula desde los contadores en **cada refresco** y no está
grabada, así que lo que la mascota *es* puede cambiar de una línea a la
siguiente. Lo que no puede es bajar por el árbol.

Dos hábitos se van a cero cuando revientas el contexto —`test_streak` y
`diff_streak`, las dos rachas limpias— y sin un suelo eso era una caída: una
`avispa` de nivel 6 volvía como `cazabugs`, una forma de nivel 3, con el
rótulo «nivel 6» al lado. `pet.json` guarda ahora en `form_seen` el peldaño más
alto pisado, y `pet.Save` lo anota **en cada escritura**, de modo que ningún
camino puede persistir una mascota y olvidarse de dónde está.

La regla es que una forma **no se cae**: se mueve en lateral, hacia arriba, o
de rama.

| desde | pasa a | por qué |
| --- | --- | --- |
| `exterminador` | `sabueso` | mismo peldaño, el otro hábito |
| `exterminador` | `avispa` | hacia arriba, su título |
| `exterminador` | `cazabugs` | **nunca**: sería bajar de peldaño |
| `avispa` | `sabueso` | **nunca**: mismo oficio, es la racha cayéndose |
| `avispa` | `cirujano` | sí: otro oficio, y la marca de allí está ganada |

Las dos últimas filas parecen la misma bajada de 6 a 5 y no lo son, así que el
suelo pregunta **por qué** salió más bajo. Si el oficio es el mismo, se ha caído
un hábito y eso es justo lo que el suelo para. Si el oficio ha cambiado, la
mascota se ha movido de rama y la marca de la rama nueva está pagada: enseñar un
título de una rama que ya no pisa dice menos que enseñar lo que es hoy. Por
debajo del peldaño 5 no pasa nada en ninguna dirección — un camino que devuelve
un oficio pelado es xp cayendo o una rama sin nada ganado todavía.

Dos consecuencias que conviene conocer:

- **Cambiar de temperamento sí cambia la forma**, en cuanto la rama nueva
  tenga una marca ganada — y también si ya llevas título. Lo que no hace es
  dejarte en el oficio pelado mientras no haya nada ganado allí.
- **El nivel sí puede bajar**, y la forma no. Son dos hechos distintos: la
  forma es una marca de agua y el nivel es la xp de hoy, que cae al reventar el
  contexto (−15) y mientras la mascota pasa hambre. Así que `avispa nivel 5` es
  una pareja que se puede ver, y no es un fallo.

## La comida

| Comida | XP | Hambre | Tope |
| --- | --- | --- | --- |
| tests en verde | **+15** | −4 | una cada hora, y con un cambio detrás |
| commit hecho | **+12** | −3 | — |
| `/compact` | **+8** | −3 | — |
| tarea del plan cerrada | **+6** | −1 | — |
| `/feed` | **+3** | −2 | uno cada 4 h |
| contexto al 100% | **−15** | — | — |
| cada hora a hambre 10 | **−1** | — | — |

Y al cerrar sesión, sin XP de por medio: pico bajo del 60% suma `ctx_low`
(hacia `pulcro`), pico por encima del 85% suma `impulsive` (hacia `brasa`).

**Por qué la suite verde tiene freno.** Era la comida más grande de la tabla y
la única sin ningún tope, así que era lo único que compensaba farmear: correr
la suite en bucle daba +15 cada pocos segundos —120 XP en ocho minutos, medido
en una sesión real— y con eso el techo y el drenaje eran decoración. Nada que
se repita en nueve segundos puede valer un quinceavo de nivel.

Van dos frenos, porque resuelven cosas distintas:

- **Una cada hora.** No es un número al azar: el lienzo presupuestaba el nivel 5
  en «una semana de uso normal», o sea unos 128 XP al día, y ocho suites verdes
  en una jornada son exactamente eso. El freno sigue calibrado ahí; lo que ha
  cambiado es el destino, no el ritmo — el nivel 5 se alejó a propósito para
  que la bifurcación del hábito tenga tiempo de decidirse.
- **Y con un cambio detrás.** Una suite que pasa sin que hayas editado nada no
  es trabajo, es la misma suite otra vez. El hook ya sabía qué herramientas se
  usan, así que le basta con recordar si hubo un `Edit` desde la última vez que
  cobró. El ciclo rojo → verde del sabueso se apunta igual aunque la suite no
  pague: reproducir un fallo cuenta por sí solo.

Cada comida lleva **su propio reloj** (`meals` en `pet.json`). Antes había uno
solo, `fed_at`, que bastaba mientras `/feed` era la única con espera; dos
comidas con freno se habrían amordazado la una a la otra.

El **hambre** sube +1 por hora sin comer, hasta 10. A partir de 7 los ojos se
apagan y la mascota pide comida en la statusline. Al llegar a 10 deja de ser un
aviso y **empieza a costar 1 XP por hora**, que es la única forma que tiene la
mascota de perder terreno solo. **Nunca muere**: por abajo se queda en larva, que
es una forma, no una tumba.

Reventar el contexto resta 15 XP y rompe las rachas limpias.

### Por qué el nivel sí baja

El diseño original decía que el nivel nunca baja, y con esa regla una mascota que
llegaba al tope se quedaba ahí para siempre: no había nada que ganar ni nada
que perder. La escalera terminaba y el tamagotchi dejaba de serlo.

Dos cambios lo corrigen sin tocar el árbol, porque la maquinaria de bajar ya
estaba entera —`LevelFor` sigue a la XP en las dos direcciones, y con el nivel
baja la forma—; lo que faltaba era que algo restase de verdad:

- **La XP tiene techo**: `XPCeiling`, el último umbral más un tramo de nivel 1.
  Sin él la XP era un foso. Con 1641 puntos y el tope en 900 hacían falta
  cincuenta contextos reventados para bajar un nivel, así que cualquier castigo
  se ahogaba en el colchón antes de significar nada.
- **El hambre al tope drena.** Medio día fuera no cuesta nada; a los **dos días
  y medio** se pierde el último nivel. Ese número no se mueve al tocar la
  escalera, porque el techo se define *relativo* al último umbral: son las 60
  horas del tramo de nivel 1, siempre.

  Caer del todo, en cambio, sí escala con la escalera, y ahora son **190 días**
  —antes eran unos 80—. La doc decía «seis semanas» y ya llevaba tiempo sin ser
  verdad. Si 190 días parece demasiado indulgente, lo que hay que mover es
  `StarveXP`, no el techo.

Las cifras viven en `StarveXP` y `XPCeiling`, y hay un test
(`TestTheCostOfNeglectIsWhatWeMeantItToBe`) que discute con quien las mueva.

## Qué alimenta cada contador

El hook (`ccpet hook`) y la propia statusline traducen lo que haces en
contadores. **Las 41 evoluciones son alcanzables**: la raíz, los tres
temperamentos, los siete oficios, las catorce marcas, los catorce títulos y las
dos secretas. Y siguen siéndolo con la mascota ya crecida, que es lo que
`TestEveryFormIsReachableFromAVeteran` fija: en cada bifurcación gana el hábito
que más lejos ha llegado *respecto a lo que pide* —su umbral en el nivel 5, su
escala en los niveles 2 y 3— y no el primero que cruzó una línea, así que
ninguna puerta se cierra a tu espalda.

| Contador | Se llena con | Quién lo ve |
| --- | --- | --- |
| `methodical` | commits y `/compact` | hook |
| `inquisitive` | tests y tareas del plan | hook |
| `impulsive`, `ctx_maxed` | el pico de contexto de la sesión (85 / 95) | statusline → `SessionEnd` |
| `diffs`, `diff_streak` | commits (la racha se rompe al reventar) | hook |
| `tests`, `test_streak` | tests en verde (íd.) | hook |
| `widest_commit` | el `N files changed` más alto | hook |
| `longest_plan` | el plan más largo cerrado del todo | hook |
| `plans_before_code` | un plan de 3+ tareas escrito antes de editar nada | hook |
| `single_tool_tasks` | tareas cerradas usando una sola herramienta | hook |
| `repro_before_fix` | un test rojo seguido de uno verde | hook |
| `docs_days` | días seguidos con un commit de docs o de limpieza | hook + `git show --numstat` |
| `bypass_turns` | prompts nuevos con los permisos en bypass | statusline + transcript |
| `ctx_low`, `sessions_under_40` | el pico de contexto de la sesión | statusline → `SessionEnd` |
| `short_sessions`, `sessions_15min`, `long_sessions`, `sessions_4h` | la duración de la sesión | íd. |
| `ctx100_sessions` | tocar el 100% de contexto (la tercera muesca) | íd. |
| `same_repo_days` | días seguidos cerrando sesión en el mismo repo | íd. |

Tres datos **solo los ve la statusline**, porque no llegan a ningún hook: el uso
de contexto, el modo de permisos y el ritmo de tokens. Los va dejando en su
fichero temporal y el hook de `SessionEnd` los convierte en contadores.

El **modo de permisos** merece una nota: no viene en el payload de la statusline,
pero sí en el transcript, cuya ruta sí llega. Se lee la cola del fichero (32 KB,
0,02 ms) buscando el último `permissionMode`. Es la única forma de que `gremlin`
—«30 turnos con permisos en bypass»— sea alcanzable.

## Cómo se deducen las cosas que el CLI no dice

Dos comidas y cuatro marcas salen de **deducir**, no de un dato que el CLI
exponga. La regla al escribirlas ha sido siempre la misma: **preferir perderse
una comida a inventarse una.**

**Tests en verde.** Tres capas, de más dura a más blanda:

1. Un `is_error` del CLI manda sobre todo: equivale al código de salida y es el
   único dato duro que hay.
2. Los patrones de rojo se buscan **solo en las últimas doce líneas**, que es
   donde va el resumen. Buscarlos en toda la salida hacía que un test llamado
   `test_login_failed` tiñera de rojo una suite verde.
3. No hace falta un patrón de verde. Si el comando era un runner y salió bien,
   cuenta — así entran los runners que no están en la lista.

Para reconocer el runner hay una lista larga (pytest, jest, vitest, go, cargo,
phpunit, rspec, mvn, gradle, dotnet, swift, flutter, mix, make/just/task…), un
último recurso por el **nombre del ejecutable** (`run-tests.sh`, `testear.sh`,
`bin/spec` cuentan; el `test` de shell no, que es una comparación de ficheros), y
`PET_TEST_RUNNERS` para meter un regex propio.

Y lo más importante: el runner tiene que estar **en posición de comando**, no en
cualquier parte del texto. El comando se parte por los operadores de shell
(`;`, `&&`, `||`, `|`) y cada trozo se mira por su principio, saltándose las
asignaciones de entorno y el `sudo`. Sin eso, un `echo "lanza pytest"` o un
`grep -rn "go test"` contaban como suite verde — el mismo fallo que ya tenía la
detección de commits, arreglado aquí de la misma manera.

**Commit hecho.** `git … commit` tiene que estar al principio del comando o
detrás de un operador de shell. Sin ese anclaje, un `grep -rn "git commit"`
contaba como commit.

**Las cuatro marcas deducidas.** Ninguna adivina intenciones: todas miran un
hecho comprobable que se le parece mucho.

| Marca | Lo que pide el diseño | Lo que se mira de verdad |
| --- | --- | --- |
| `bloodhound` | repro antes del fix | un test rojo seguido de uno verde en la misma sesión |
| `oraculo` | planes escritos antes de tocar código | un `TodoWrite` de 3+ tareas antes del primer `Edit`/`Write` |
| `gardener` | docs y limpieza dos días seguidos | `git show --numstat` del commit: mayoría de `.md`/`docs/`, o un borrado grande |
| `sniper` | tareas cerradas con una sola herramienta | herramientas distintas usadas entre dos tareas cerradas |

Son aproximaciones y se equivocan: un rojo por un fallo de red y un verde después
cuentan como repro→fix aunque no arreglaras nada. Pero se equivocan **por lo
bajo** casi siempre, y ninguna se inventa un hecho que no haya ocurrido.

## Los mandos

```bash
pet                    # el panel: nivel, evolución, xp, hambre, racha y la comida de hoy
pet feed               # +3 xp, hambre −2, uno cada cuatro horas
pet count <c> [n]      # suma a un contador de comportamiento
pet record <c> <v>     # guarda el máximo de un contador
```

Instalados como `/pet` y `/feed` desde `scripts/install.sh`.

Para empezar de cero: `rm ~/.claude/pet.json`. Para soltar solo las ramas
defendidas y dejar que se recalculen: quitar la clave `branch`.

---

Ver también el [README](../../README.md) para las bandas y la paleta, y
[vitals.md](vitals.md) para la otra capa, la del momento.
