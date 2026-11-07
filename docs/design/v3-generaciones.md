# Mecánica v3: generaciones

> **Propuesta, no implementada.** Borrador del 7-nov-2026 para revisar antes de tocar
> código. Lo que está medido lleva `[medido]` y la invocación; lo que no, `[hipotesis]`.

## El problema

La mascota tiene 41 formas y quien la usa ve muy pocas.

- [medido] Una mascota real, tras dos meses de uso diario, ha llevado **5 formas de 41**
  (chispa, pauta, refactor, cirujano, bisturí). Puede haber cambiado de rama alguna vez al
  principio, pero `pet.json` solo guarda la forma más alta y no queda rastro.
- [medido] Está en el techo de XP (4560) y en su forma final. Comer ya no le cambia nada.
  Fin del juego.

No es mala suerte. El diseño lo produce de tres maneras:

1. **Las ramas son excluyentes.** Una mascota recorre un linaje de 6 formas como mucho.
   El 85 % del contenido es invisible por construcción.
2. **Las ramas las deciden contadores de toda la vida, y esos contadores solo suben.**
   [medido] Esa mascota lleva `methodical` 1036. Ningún hábito nuevo lo alcanza. Su rama
   quedó decidida en las primeras semanas y lo que haga hoy no cuenta.
3. **La carrera de las ramas está sesgada por el freno de XP.** El hábito `tests` solo
   sube cuando la comida *paga*, y la suite verde paga como mucho una vez por hora.
   Los commits no tienen freno. [medido] En los transcripts de octubre hay 451 ejecuciones
   de tests; en `pet.json`, el contador `tests` de toda la vida es 175.

Y lo que la mascota no ve, pero existe: el trabajo de esa persona **sí cambia**.
[medido] La misma aproximación que usan las ramas, semana a semana (script en el
apartado «Cómo se midió»):

| semana | sesiones | commits | tests | rama que ganaría |
| --- | --- | --- | --- | --- |
| 2026-W42 | 51 | 222 | 1 | pauta / refactor |
| 2026-W43 | 73 | 182 | 87 | pauta / pulcro |
| 2026-W44 | 100 | 127 | 190 | sonda / cazabugs |
| 2026-W45 | 80 | 63 | 173 | sonda / cazabugs |

Lleva dos semanas trabajando como cazabugs, y la mascota sigue enseñando bisturí.

## Lo que enseñan los v-pets

Tamagotchi y los Digimon V-Pet llevan casi treinta años resolviendo esto mismo, y hacen
dos cosas que este diseño no hace:

- **Lo que decide cada evolución es lo que pasó *en esa etapa*.** En los V-Pet de
  Digimon, errores de cuidado, entrenamientos y combates se ponen a cero en cada
  evolución ([withthewill, Ver. 1](https://lcd.withthewill.net/VpetEvo1)). En la mayoría de
  Tamagotchi pasa lo mismo con los errores de cuidado
  ([Tamagotchi Wiki, Care](https://tamagotchi.fandom.com/wiki/Care)). Así la forma adulta
  depende de cómo la cuidaste mientras era joven, no de toda su vida.
- **La vida termina y empieza otra.** Cada generación es una partida nueva, y como el
  cuidado cambia, salen formas distintas. Es eso lo que llena la colección, no la
  cantidad de contenido.

Y una tercera, de los Digimon: **hay una ruta por defecto**. Si no cumples ninguna
condición, la mascota sigue por ella. Nunca se queda atascada.

[hipotesis] En una búsqueda rápida, las mascotas para Claude Code que hay publicadas
(familiar, claude-code-tamagotchi, catcode, otra llamada también `ccpet`) reaccionan al
uso, y ninguna parece evolucionar según los hábitos. No he leído su código.

## La propuesta

### 1. Cada bifurcación la deciden los hábitos de la etapa

- Además de los contadores de vida, la mascota lleva **contadores de etapa**, que se
  ponen a cero cada vez que evoluciona.
- La bifurcación se decide **una vez, al cruzar el nivel**, con los contadores de esa
  etapa escalados por `BranchScale`, igual que hoy. Después ya no se mueve.
- Esto **elimina** la rama defendida (`branch`, `BranchMargin`, `RememberBranch`). Ese
  mecanismo existía para que la forma no cambiara de rama cada tarde, y si la decisión se
  toma en un momento y se congela, ese problema no aparece.
- La marca del nivel 5 se decide igual: al cruzar, con el hábito más maduro de la etapa.
- Si una etapa termina sin hábitos (todo a cero), se sigue la **ruta por defecto**, el
  primer hijo en el orden del árbol, como hoy en los empates.

Los contadores de vida se quedan. Los usan el panel, las estadísticas y las condiciones
de las marcas que piden cantidad («20 diffs», «5 sesiones»).

### 2. Los hábitos cuentan lo que pasa, no lo que paga

Una suite verde suma al hábito siempre. El freno de una hora sigue limitando la XP, pero
no la carrera. Con eso desaparece el sesgo del punto 3 del problema.

### 3. Generaciones y álbum

- Cuando la mascota llega a su **título**, la generación está completa. Desde ahí pone un
  huevo, y una mascota nueva empieza en chispa con los contadores de etapa a cero.
- El **álbum** guarda cada forma llevada (no solo los títulos), con la generación y la
  fecha. Las que faltan salen en silueta, con la pista del hábito que lleva hasta ellas.
  `/pet album` lo enseña.
- Las secretas (fénix, quimera) siguen igual y entran en el álbum como las demás.
- La racha diaria, el hambre y los contadores de vida **pasan de generación en
  generación**. Lo que se renueva es la forma, no tu historial.

### 4. Una generación tiene que caber en dos semanas

[hipotesis] Hoy el título está a 4500 XP. Con unos 128 XP en un día normal, eso son unas
cinco semanas, y el nivel 5 solo ya son 1600 XP. Para ver formas hay que llegar a
ellas. Propuesta inicial, a calibrar con la simulación:

| nivel | hoy | v3 |
| --- | --- | --- |
| 2 temperamento | 60 | 60 |
| 3 oficio | 180 | 200 |
| 4 asentado | 400 | 450 |
| 5 marca | 2000 | 900 |
| 6 título | 4500 | 1500 |

Unos 12 días de uso normal por generación. Cada etapa dura lo suficiente para que el
hábito de esos días decida, y la mascota ya no se queda sin juego.

## Lo que se rompe del diseño actual, a propósito

| hoy | v3 |
| --- | --- |
| la rama se recalcula en cada refresco y se defiende con un margen | se decide una vez al cruzar el nivel |
| «una forma no baja de escalón» (`form_seen`) | sigue valiendo dentro de una generación; al poner huevo se vuelve a chispa |
| la XP tiene techo y el título es el final | el título cierra la generación |
| el hambre al tope resta XP y puede bajar el nivel | igual, pero el nivel no baja de la etapa ya alcanzada en esta generación [hipotesis: confirmar] |

## Migración

Un `pet.json` de hoy es la **generación 1, completa o en curso**:

- Su linaje (sacado de `form_seen`) entra en el álbum. La mascota medida aporta chispa,
  pauta, refactor, cirujano y bisturí.
- Si ya está en el título, pone huevo en la primera carga y empieza la generación 2.
- Si no, sigue su generación con contadores de etapa a cero. La siguiente bifurcación la
  decide lo que haga a partir de ese momento.
- Así deja de importar si era bisturí o abad: bisturí está en el álbum, y la forma
  siguiente la decide el trabajo de las próximas semanas.

## Preguntas abiertas

1. ¿El huevo sale solo al llegar al título, o el título se queda unos días para lucirlo
   y luego `/pet huevo`? Propuesta: se queda **3 días de uso** y después pone huevo solo.
2. ¿Hace falta el árbol de 97 formas? Propuesta: **no**. Primero, que se vean las 41.
3. ¿Contadores de etapa también por repo? Por proyecto, el trabajo varía mucho
   (`aluvion` sale refactor y `black-hole` cazabugs), pero una mascota por repo se lleva
   la continuidad. Propuesta: **no en v3**, pero el álbum puede anotar en qué repo se
   ganó cada forma.

## Cómo se midió

[medido] 7-nov-2026, sobre `~/.claude/projects/*/*.jsonl` (sesiones principales, sin
subagentes), con `python3 -I docs/design/v3-semanas.py`. Aproximaciones: commits = `git … commit` en
posición de comando; tests = ejecutable de runner en posición de comando (pytest, go
test, tq, …); tareas = `TodoWrite`/`TaskUpdate` completadas. No mide `impulsive`, porque
el pico de contexto no está en el transcript.

AVISO: Solo hay transcripts desde la semana 41 (y una sesión suelta de la 29), así que no
se puede medir septiembre. [hipotesis] Los ha borrado la limpieza periódica de Claude Code;
no lo he comprobado.
