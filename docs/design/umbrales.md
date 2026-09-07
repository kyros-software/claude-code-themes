# El árbol de 97 formas: puertas y umbrales

> **Trabajo de diseño, no implementado.** El código sigue con sus 41 formas. Esto es
> el árbol del lienzo de Claude Design contrastado contra el runtime, con su defecto
> encontrado y el arreglo verificado.
>
> **Ya no hay que creerse esta página: se ejecuta.** Las puertas viven en
> `internal/pet/testdata/PUERTAS-97.json` y las comprueba
> `internal/pet/reach97_test.go`. Si alguien cambia una puerta y rompe una marca, la
> suite se pone roja con el nombre de la marca.
>
> Traspaso original: <https://claude.ai/code/artifact/e15cd05f-cb01-4e68-a336-85c61100faee>

## La regla, entera

Sin esto no se puede rehacer ninguna cuenta de las de abajo, y la versión anterior de
este documento no la traía: remitía al lienzo, que ya está agotado.

**Nivel 2, el temperamento.** Gana el más alto de tres contadores, empate por orden de
lista. `evolution.go:97` (`BranchBy`) y `evolution.go:437` (`topBranch`).

| temperamento | contador | oficios que cuelgan |
| --- | --- | --- |
| `pauta` | `methodical` | refactor, pulcro |
| `sonda` | `inquisitive` | cazabugs, arquitecto |
| `brasa` | `impulsive` | velocista, maratón, salvaje |

**Nivel 3, el oficio.** Misma regla entre los hermanos de ese temperamento, cada uno
con su propio contador. Ese contador queda **fuera** de las seis puertas del oficio,
que es la exclusión que el lienzo describe como «los nueve que no ganaron el oficio».

| oficio | `refactor` | `pulcro` | `cazabugs` | `arquitecto` | `velocista` | `maratón` | `salvaje` |
| --- | --- | --- | --- | --- | --- | --- | --- |
| contador | `diffs` | `ctx_low` | `tests` | `plans` | `short_sessions` | `long_sessions` | `ctx_maxed` |

**Nivel 5, la marca.** Aquí es donde el lienzo y este documento discrepan. El lienzo
quiere una carrera de cuentas crudas; lo que se propone es una carrera de ratios con
umbral por marca.

Las dos consecuencias que se olvidan si no está escrito: **para estar en un oficio ya
ganaste una carrera**, así que dentro de él hay desigualdades garantizadas entre
contadores; y **el desempate premia a la primera de la lista**, así que el orden en
que el lienzo dibuja las seis marcas es información, no maquetación.

## Los diez contadores no son diez señales

Son diez lecturas de seis cosas. `internal/pet/feeding.go:58` ata dos pares y
`internal/hook/app.go:411-460` anida el resto:

```
comida "tests"   -> inquisitive, tests   ⟹  inquisitive = tests + plans
comida "task"    -> inquisitive, plans
comida "commit"  -> methodical, diffs    ⟹  methodical  = diffs + compacts
comida "compact" -> methodical
pico < 40        -> sessions_under_40    ⟹  ctx_low     ≥ sessions_under_40
pico < 60        -> ctx_low
pico ≥ 85        -> impulsive            ⟹  impulsive ≥ ctx_maxed ≥ ctx100_sessions
pico ≥ 95        -> ctx_maxed
pico = 100       -> ctx100_sessions
sesión < 15 min  -> short_sessions, sessions_15min   ⟹  son el MISMO número
sesión ≥ 90 min  -> long_sessions        ⟹  long_sessions ≥ sessions_4h
sesión ≥ 4 h     -> long_sessions, sessions_4h
```

Las tres primeras salen de las comidas; las demás, del cierre de sesión. Una carrera
entre una suma y su sumando tiene ganador antes de empezar.

Y hay algo peor que las colisiones sueltas: **los diez contadores que el lienzo usa
para elegir marca son exactamente los diez que ya decidieron el temperamento y el
oficio** — tres de nivel 2 más siete de nivel 3. El nivel 5 no aporta ninguna señal
nueva; reparte otra vez las que ya gastó. Ahí está la raíz, y ninguna reasignación de
puertas la quita del todo.

## El defecto, en dos varas

| vara | alcanzables | muertas |
| --- | ---: | ---: |
| gana incluso empatando (el desempate por orden le regala la marca) | 21 | 21 |
| gana **sin empatar** | 15 | 27 |

Las 21 muertas con cualquier vara: `andamio`, `avalancha`, `cepo`, `cimiento`,
`cristal`, `erizo`, `flecha`, `francotirador`, `fuente`, `grieta`, `incendio`,
`injerto`, `jardinero`, `kraken`, `lienzo`, `lima`, `linterna`, `muelle`, `oráculo`,
`relámpago`, `sabueso`.

Las 6 que solo viven de un empate exacto —`cirujano`, `tejedor`, `buey`, `caravana`,
`muro`, `reloj`— piden que dos contadores coincidan al entero (`compacts` a cero y
`plans` igualando a `methodical`, por ejemplo). En uso real eso no pasa: el defecto
es de 27, no de 21.

Esto importa porque la versión anterior comparaba «21 antes» con «42 después» y no
eran la misma métrica: el 42 se exigió sin empates. Con la misma vara a los dos lados,
la mejora es **de 15 a 42**.

## El arreglo: dos piezas, y una ya está construida

**Uno — quince puertas cambian.** Ningún oficio conserva un contador junto a otro que
lo contiene. No depende de ningún ritmo: es álgebra.

**Dos — cada marca pide un umbral, y la carrera es de ratios** (`contador ÷ lo que
pide`). El umbral no es una puerta que cruzar: es el conversor de unidades que permite
comparar `bypass_turns`, que va por turno, con `ctx100_sessions`, que sube una vez por
sesión reventada.

La pieza dos **no hay que diseñarla: el runtime ya la hace**. `ripestMark`
(`evolution.go:343`) elige por ratio, `Unlocks` (`evolution.go:111`) guarda el par
contador/umbral de cada marca y `Mark.Share` lo pinta en la banda 4. Se puso ahí por
este mismo defecto en el árbol de 41 —su comentario cuenta cómo ocho de las cuarenta y
una formas habían dejado de existir— y lo que falta es rellenar `Unlocks` con 42
entradas, no inventar mecánica.

El lienzo rechaza los umbrales porque «un bicho que no cumpliera ninguno se quedaría
sin oficio». Es cierto de una puerta dura y falso de una carrera de ratios: el máximo
de un ratio existe siempre, igual que el de una cuenta.

## Los ritmos, medidos

366 transcripts de `~/.claude/projects`, 31 días, con los patrones de detección del
propio hook y sus cooldowns aplicados. *(Medición heredada de la sesión anterior; no
se ha vuelto a correr. Lo que sí se comprobó es que el `pet.json` de esta máquina
sigue con `plans`, `impulsive`, `ctx_maxed` y `ctx100_sessions` a cero, que es lo que
obliga a estimar cuatro umbrales.)*

| Señal | Por semana | Fuente | Alimenta |
| --- | ---: | --- | --- |
| turnos de usuario | 682 | medido · transcripts | — |
| turnos en bypass | 651 | medido · pet.json, 95% del total | `bypass_turns` |
| sesiones | 34 | medido · 134 con marcas de tiempo | — |
| suites verdes (tras cooldown) | 53,3 | medido | `tests` `inquisitive` |
| commits | 46,3 | medido | `diffs` `methodical` |
| sesiones >90 min | 22,8 | medido · 68% | `long_sessions` |
| sesiones ≥4 h | ~8 | **derivado** · 6/17 del `pet.json` | `sessions_4h` |
| compactados | 5,4 | medido | `methodical` |
| sesiones <15 min | 5,0 | medido · 15% | `short_sessions` |
| tareas de plan cerradas | 0 | medido · cero TodoWrite en 366 ficheros | `plans` |
| sesiones con pico ≥85% | 7 | **estimado** · el pet está a 0 | `impulsive` |
| sesiones con pico ≥95% | 3 | **estimado** · el pet está a 0 | `ctx_maxed` |
| sesiones al 100% | 1 | **estimado** · el pet está a 0 | `ctx100_sessions` |

A ese ritmo son **1.398 xp/semana**, así que el nivel 5 (2000 xp) llega en **10
días**. Cada umbral es el valor esperado de su contador al llegar ahí.

**Comprobación cruzada.** `19 diffs / 46,3` = 2,9 días; `23 tests / 53,3` = 3,0;
`12 long_sessions / 22,8` = 3,7, contra un `streak: 3` en el `pet.json` de entonces. Y
el bypass sale por dos caminos independientes que concuerdan al 5%.

## Las 42 marcas

`~~tachado~~` es la puerta del lienzo; en negrita, la nueva. Los títulos se
comprobaron uno a uno contra los padres de `ATLAS-97.json`: los 42 cuadran.

| oficio | marca | puerta | pide | título |
| --- | --- | --- | ---: | --- |
| refactor | `cirujano` | `ctx_low` | 80 | `bisturí` |
| refactor | `tejedor` | `plans` | 29 | `telar` |
| refactor | `molde` | ~~`methodical`~~ → **`impulsive`** | 10 | `imprenta` |
| refactor | `lima` | `tests` | 76 | `espejo` |
| refactor | `injerto` | ~~`inquisitive`~~ → **`short_sessions`** | 7 | `raíz` |
| refactor | `tijera` | `long_sessions` | 33 | `guillotina` |
| pulcro | `monje` | ~~`methodical`~~ → **`impulsive`** | 10 | `abad` |
| pulcro | `jardinero` | `plans` | 29 | `bosque` |
| pulcro | `fuente` | `diffs` | 66 | `acueducto` |
| pulcro | `cristal` | `tests` | 76 | `prisma` |
| pulcro | `nieve` | `short_sessions` | 7 | `ventisca` |
| pulcro | `lienzo` | ~~`inquisitive`~~ → **`long_sessions`** | 33 | `mural` |
| cazabugs | `sabueso` | `plans` | 29 | `lobo` |
| cazabugs | `exterminador` | ~~`inquisitive`~~ → **`impulsive`** | 10 | `avispa` |
| cazabugs | `cepo` | ~~`diffs`~~ → **`short_sessions`** | 7 | `red` |
| cazabugs | `linterna` | `methodical` | 74 | `faro` |
| cazabugs | `anzuelo` | `ctx_low` | 80 | `arpón` |
| cazabugs | `lupa` | `long_sessions` | 33 | `microscopio` |
| arquitecto | `cartógrafo` | ~~`inquisitive`~~ → **`impulsive`** | 10 | `atlas` |
| arquitecto | `oráculo` | `tests` | 76 | `esfinge` |
| arquitecto | `andamio` | `methodical` | 74 | `catedral` |
| arquitecto | `brújula` | `ctx_low` | 80 | `sextante` |
| arquitecto | `cimiento` | ~~`diffs`~~ → **`short_sessions`** | 7 | `muralla` |
| arquitecto | `maqueta` | `long_sessions` | 33 | `ciudad` |
| velocista | `relámpago` | ~~`diffs`~~ → **`long_sessions`** | 33 | `tormenta` |
| velocista | `francotirador` | `tests` | 76 | `halcón` |
| velocista | `flecha` | `methodical` | 74 | `saeta` |
| velocista | `muelle` | `plans` | 29 | `resorte` |
| velocista | `chispazo` | ~~`impulsive`~~ → **`ctx_maxed`** | 4 | `descarga` |
| velocista | `patín` | `ctx_low` | 80 | `cohete` |
| maratón | `buey` | ~~`diffs`~~ → **`short_sessions`** | 7 | `mamut` |
| maratón | `topo` | `methodical` | 74 | `gusano` |
| maratón | `ancla` | `ctx_low` | 80 | `puerto` |
| maratón | `caravana` | `plans` | 29 | `legión` |
| maratón | `muro` | `tests` | 76 | `bastión` |
| maratón | `reloj` | ~~`inquisitive`~~ → **`ctx_maxed`** | 4 | `calendario` |
| salvaje | `gremlin` | ~~`impulsive`~~ → **`bypass_turns`** | 931 | `diablo` |
| salvaje | `kraken` | `long_sessions` | 33 | `leviatán` |
| salvaje | `avalancha` | ~~`diffs`~~ → **`ctx100_sessions`** | 2 | `glaciar` |
| salvaje | `erizo` | `short_sessions` | 7 | `espina` |
| salvaje | `incendio` | ~~`methodical`~~ → **`sessions_4h`** | 12 | `volcán` |
| salvaje | `grieta` | `tests` | 76 | `abismo` |

### Por qué `salvaje` no puede llevar `impulsive`

En su propia rama `impulsive` es el contador grande por construcción: supera a
`methodical` y a `inquisitive` porque ganó el temperamento, y contiene a `ctx_maxed`,
que a su vez supera a `short_sessions` y `long_sessions` porque ganó el oficio. Casi
todo lo que puede pedir una marca de `salvaje` está debajo de él.

Con `gremlin` pidiendo `impulsive` a 10 —el umbral más bajo del oficio— eso mata a
`kraken` (`long_sessions/33`) y a `grieta` (`tests/76`), y sube el umbral no arregla
nada: si sube lo bastante para que ganen, `gremlin` deja de ganar nunca. No hay valor
que equilibre. Es el mismo defecto del lienzo, reintroducido en el arreglo.

El runtime ya lo había resuelto en la rama equivalente: `Unlocks` abre las dos marcas
de `feral` con `bypass_turns` y `ctx100_sessions`, nunca con `impulsive`. Aquí se hace
lo mismo, y de paso `gremlin` recupera la puerta que el código ya le da.

`incendio` se queda entonces sin `bypass_turns` y toma `sessions_4h`, que el hook ya
alimenta (`app.go:457`) y que arde con la narrativa del volcán. Su umbral es el único
**derivado** y no medido: 33 `long_sessions` por la proporción 6/17 que marca el
`pet.json` de esta máquina.

## Lo que no está sólido

- **La raíz sigue ahí.** El nivel 5 se alimenta de los mismos diez contadores que ya
  se gastaron arriba. Este arreglo los recoloca para que ninguna colisión sea fatal,
  y lo verifica un test; pero cada casilla depende de la relación entre dos umbrales,
  y los umbrales salen de ritmos medidos, no son libres. **La alternativa de fondo es
  la que el runtime ya tomó**: 14 contadores propios de marca (`diff_streak`,
  `repro_before_fix`, `sessions_under_40`, `widest_commit`, `longest_plan`…), ninguno
  de los cuales decide rama alguna. Con esos, el problema no existe por construcción.
  Cambiar las 42 puertas a contadores propios es rehacer el nivel 5 del lienzo, y es
  una decisión de diseño abierta, no un arreglo.
- **Cuatro umbrales son estimados** —`impulsive`, `ctx_maxed`, `ctx100_sessions`,
  `plans`— y uno derivado —`sessions_4h`—. Este perfil los tiene a cero.
- **El reparto es modelo, no medida.** Que las 42 sean alcanzables está verificado;
  con qué frecuencia sale cada una depende del perfil de usuario que se suponga, y ahí
  no hay dato. Un muestreo uniforme da repartos muy desiguales (`chispazo` 99% de
  `velocista`); eso no describe uso real, pero avisa de que el equilibrio no está
  demostrado.
- **Dos factores de título son invención, no dato**: los de `methodical` e
  `impulsive`, que no tienen pariente entre los catorce que el lienzo calibró. Llevan la
  mediana de los otros doce. Si aparece el lienzo con sus 42 factores, esos dos son los
  primeros que hay que sustituir.
- **`gremlin` ya no es la marca por defecto de la rama feral**, porque `bypass_turns`
  a 931 no se cruza de casualidad. Quién ocupa ese sitio es diseño, no aritmética.

## Qué está verificado: 95 de las 97

| formas | cuántas | estado |
| --- | ---: | --- |
| la raíz `chispa` | 1 | trivial |
| temperamentos | 3 | verificado |
| oficios | 7 | verificado |
| marcas | 42 | **verificado, 42/42 sin empates** |
| títulos | 42 | **verificado, con el número puesto** |
| secretas (`fénix`, `quimera`) | 2 | fuera de este árbol — regla propia, ya en el runtime |

## Los 42 umbrales de título

Un título no compite con nadie: va detrás de su marca y pide *más del mismo contador*
(`TitleUnlock`, `evolution.go:182`). La pregunta no es quién gana, es si se puede
seguir subiendo ese contador **sin perder la marca por el camino** — y eso no es
gratis, porque cada contador alimenta algo más. Llegar a `volcán` son 40
`sessions_4h`, que son 40 `long_sessions`, que es la puerta donde espera `kraken`.
Verificado marca por marca: se puede.

El número es lo que no estaba en ninguna parte. **Diez de los doce factores se heredan
del lienzo**, que ya calibró un título para ese mismo contador o para un gemelo
verificado; los otros dos llevan la mediana de los doce que el lienzo sí fijó.

| contador | marca | título | factor | de dónde sale el factor |
| --- | ---: | ---: | ---: | --- |
| `short_sessions` | 7 | 21 | ×3 | `sessions_15min` es el **mismo número** (`app.go:452-453`); `storm` pide ×3 |
| `sessions_4h` | 12 | 40 | ×3,33 | mismo contador; `mammoth` pide ×3,33 |
| `ctx100_sessions` | 2 | 7 | ×3,33 | mismo contador; `leviathan` pide ×3,33 |
| `bypass_turns` | 931 | 3.100 | ×3,33 | mismo contador; el lienzo pedía 100 sobre los 30 de `gremlin` |
| `ctx_low` | 80 | 240 | ×3 | `sessions_under_40` es el mismo pico con otro corte; `abbot` pide ×3 |
| `ctx_maxed` | 4 | 13 | ×3,33 | misma familia de picos; `leviathan` pide ×3,33 |
| `long_sessions` | 33 | 110 | ×3,33 | misma familia de duración; `mammoth` pide ×3,33 |
| `tests` | 76 | 255 | ×3,33 | mismo hábito que `test_streak`; `wasp` pide ×3,33 |
| `plans` | 29 | 115 | ×4 | mismo hábito que `plans_before_code`; `sphinx` pide ×4 |
| `diffs` | 66 | 165 | ×2,5 | mismo hábito que `diff_streak`; `scalpel` pide ×2,5 |
| `methodical` | 74 | 240 | ×3,23 | **sin pariente**: mediana de los doce factores del lienzo |
| `impulsive` | 10 | 30 | ×3,23 | **sin pariente**: mediana de los doce factores del lienzo |

`TitleAsks` consiguió sus catorce números del lienzo, uno por título, sustituyendo a un
multiplicador uniforme que se había inventado en ese fichero — así que escribir aquí un
multiplicador uniforme nuevo habría repuesto justo lo que aquel cambio quitó. Dos
números inventados de cuarenta y dos, los dos marcados, es el suelo al que se puede
llegar sin volver al lienzo.

Un test comprueba que ningún factor se sale del rango ×2,0–×4,5 que el lienzo gastó en
sus catorce, para que una edición futura no meta un ×10 sin que salte nada.

## Cómo se verificó

- **Alcanzabilidad** — `go test ./internal/pet/ -run 'NinetySeven|Temperament|Title'`.
  Testigo constructivo por marca: se levanta el estado mínimo que lleva al oficio y se sube el
  contador objetivo, subiendo siempre, nunca bajando. 42/42 sin empates.
- **Contraste** — tres métodos independientes coinciden en el diagnóstico (21 muertas
  con la regla del lienzo): enumeración exhaustiva, muestreo de 4 M estados y el
  testigo constructivo. La lista sale idéntica nombre por nombre.
- **Lo que cazó un falso negativo** — el primer buscador era aleatorio y dio por
  muertas dos marcas que sí son alcanzables (`incendio`, `linterna`). Por eso el test
  del repo es constructivo: la búsqueda a ciegas se equivoca hacia el lado que parece
  prudente.
- **Los títulos** — los 42 pares marca/título de la tabla se comprobaron contra los
  padres de `ATLAS-97.json`. Cuadran los 42.

## El atlas está completo

`internal/pet/testdata/ATLAS-97.json` — las 97 formas con nombre, padre, nota, color
base, rampa de siete pasos y las siete siluetas de cinco filas. Mismo esquema que el
`ATLAS.json` de 41 que usan los tests, que se deja intacto hasta que esto se
implemente.

Extraído del lienzo en cinco trozos (los ficheros completos superan el tope de 256
KiB de `DesignSync.get_file`) y verificado:

| Comprobación | Resultado |
| --- | --- |
| formas | 97 |
| estructura | todas: 7 estados × 5 filas × 9 columnas, rampa de 7 |
| árbol | 1 + 3 temperamentos + 7 oficios + 42 marcas + 42 títulos + 2 secretas |
| contra `ATLAS.json` | 41 comunes, 40 idénticas carácter por carácter |
| siluetas duplicadas | 0 — el lienzo promete «sin dos iguales» |
| rampas distintas | 10 — coincide con «Ten ramps» de `ramps.go:15` |
| variantes | 97 × 7 = 679, los dos números del lienzo (371 + 308) |

Las tres últimas se cumplen sobre datos extraídos por separado. La de las variantes se
sigue de la estructura y no es evidencia independiente del parser; lo que sí dice es
que el lienzo contaba 97 formas.

**La única discrepancia: `diablo` está redibujado.** Misma rampa y mismo color base,
pero tres de sus cinco filas cambian — los cuernos pasan de `^ ╲ ╱ ^` a `^^ ╲ ^^`, la
base de `▝▙▄█▄▟▘` a `▝▙▄▀▄▟▘` y las patas se juntan. Es un cambio de diseño posterior
al código, no un error de extracción.

## Qué queda para implementarlo

Ya no falta ningún dato. Queda el trabajo: `ATLAS.json` pasa a ser el de 97,
`Unlocks` pasa de 14 a 42 entradas con las de `PUERTAS-97.json`, `sprites.go`,
`ramps.go`, `evolution.go` y `names.go` crecen con las 56 formas nuevas, los tests que
hoy afirman 41 formas y 287 variantes pasan a 97 y 679, y hay que migrar los
`pet.json` que ya lleven una marca cuya puerta cambia.

**Un aviso sobre los nombres.** `PUERTAS-97.json` y `ATLAS-97.json` nombran las formas
en español, que es como las nombra el lienzo. Los **ids** del código son en inglés
(`monk`, `gremlin`, `feral`) y están escritos en los `pet.json` que la gente ya tiene,
así que no se renombran: `names.go` traduce. Implementar esto incluye darle id inglés a
cada una de las 56 formas nuevas y decidir el mapeo — el fichero de puertas no lo trae,
y no debe traerlo: es el dato del lienzo, no la tabla del runtime.
