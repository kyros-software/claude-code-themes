# El uso de la mascota

Qué mide exactamente la mascota de la statusline, y qué hace que pase de *fresh* a
*k.o.*

Esta es **una de las dos capas**. La vida es del momento: sube y baja con el uso
y se recupera al compactar. La otra capa, el progreso —la XP que elige en qué
evoluciona—, está en [evolution.md](evolution.md) y no baja nunca.

## Un solo número

Todo —estado, ojos, patas, cabeza, color— sale de **un número entre 0 y 100**: lo
llena que está la ventana de contexto **de esta sesión**.

```
uso = context_window.used_percentage
```

Ejemplo real: contexto al 36% → `lively`. Ni las cinco horas ni los siete días
entran: son de la **cuenta**, no de la sesión, y tienen su propio sitio en la
banda 1.

## Por qué solo el contexto

Esta es la tercera respuesta a la misma pregunta, y las dos anteriores no se
tiraron por capricho. Cada una arregló algo real y rompió otra cosa.

**Primero fue una media ponderada 50/30/20**, con un argumento razonable detrás:
el contexto es lo único que puedes gestionar en el momento —compactas, cierras
la sesión, abres otra—, así que pesa más, pero los límites también aprietan.

Lo que ese argumento no vio es que **una media diluye** justo el caso que
importa. Con la ventana llena del todo y las cuotas ociosas:

```
media:   0.5·100 + 0.3·20 + 0.2·10  =  58   →  "a gusto", turquesa
```

El contexto agotado, sin sitio para trabajar, y la mascota diciendo que está
cómodo. Eso no es una ponderación desafortunada: es el número mintiendo justo
cuando hacía falta que no lo hiciera.

**Después fue el cuello más apretado**, `max(ctx, 5h, 7d)`, que es lo que medía
la primera versión del proyecto (`statusline.sh`, commit `05bf5c7`) bajo una
frase que sigue sonando bien: *«no finge emociones; refleja el cuello más
apretado»*. Arregló la dilución de golpe y trajo un problema del que aquel
documento ya avisaba —«si el límite de 7 días va por el 95%, la mascota está
`drowning` toda la semana aunque abras la sesión con la ventana vacía»— y que se
juzgó el precio barato.

No lo era, porque el problema es peor de lo que decía ese aviso. **Las cuotas son
de la cuenta.** Todas las sesiones abiertas leen el mismo número, así que la
mascota dejaba de describir la sesión en la que vive:

```
sesión A   ventana al  6%,  5h al 81%   →  cansada
sesión B   ventana al 64%,  5h al 81%   →  cansada
```

Dos ventanas que no se parecen en nada, dos mascotas idénticos, y un `/clear` que
no cambiaba nada porque lo que gobernaba no era el contexto. La lectura no
mentía sobre la cuenta; mentía sobre **la sesión**, que es de lo que la mascota
habla.

**Y el contexto sí es una experiencia.** Una ventana llena es un Claude más
lento y más espeso, algo que notas mientras trabajas, en esta terminal, en esta
conversación. `espesa` y `cansada` describen eso. Una cuota al 81% no se nota en
ninguna respuesta: se nota cuando te corta, y para eso no hace falta una cara,
hace falta un número.

## Las cuotas no desaparecen

Siguen en la banda 1, como números, **pintados con esta misma escalera**:

```
████░░░░░░░░░░░░ 7% · 1M ctx │ xhigh │ 5h 82%  7d 21%
```

Un `5h` al 95 sale en el índigo de `drowning`, así que lo que está a punto de
pararte es el color más fuerte de la línea aunque la mascota esté verde. Eso es
todo lo que necesitan: dicen cuánto queda del día, y eso se lee en una cifra.

Las cuentas por API no reciben `rate_limits`, así que ahí no hay nada que leer —y
antes eso obligaba a que la mascota tuviera un caso especial. Ya no.

## La curva viene de la primera versión

Los umbrales no son arbitrarios ni se han tocado nunca. Son la curva de comodidad
que dibujaba `statusline.sh`, **cuadrática** —alta y plana abajo, se desploma solo
cerca del tope, porque *un 44% no es media vida*—, resuelta para el uso:

```
vida = 100 · (1 − (uso/100)²)      →      uso = 100 · √(1 − vida/100)

vida 95 → 22.36 → cap 22        vida 40 → 77.46 → cap 78
vida 80 → 44.72 → cap 45        vida 20 → 89.44 → cap 89
vida 60 → 63.25 → cap 63
```

La curva ha sobrevivido intacta a las tres entradas; lo único que se desviaba era
lo que se le metía. Hay un test que la fija
(`TestTheThresholdsAreTheFirstVersionsComfortCurve`).

## La barra de la banda 1 mide lo mismo

La barra y la mascota son **una sola medida**. Es la única disposición que no ha
fallado, y las otras dos se probaron:

- La barra medía el contexto y solo **tomaba prestado el color** del cuello: con
  el contexto al 48% y la cuota de 5h al 67 salía una barra a media asta junto a
  la palabra `espesa`, que es la lectura del 67. Dos números en la misma línea, y
  el que mandaba era el que no se veía.
- Se ascendió la barra al cuello para cerrar esa grieta, y entonces la banda
  imprimía `82% 5h` tres columnas antes de imprimir `5h 82%` otra vez, mientras
  el contexto —el único de los tres que es de esta sesión— se quedaba con un `7%`
  suelto y sin barra.

Ahora el largo, el número y el color son el contexto, y la mascota es ese mismo
contexto. No pueden discrepar porque no hay dos cosas.

El color, además, es **el cuerpo de la mascota**: la rampa de su rama en el peldaño
que elige el estado. Antes era el color de la escalera de estados, que coincidía
con la mascota en *cómo* va la sesión pero no en *quién* la está viviendo —una
escalera única para todos las mascotas, cuando desde el atlas el tono es de la
rama. Un `cazabugs` azul junto a una barra verde que significaba lo mismo.

## Dónde cae cada estado

| Uso | Estado | Ojos | Cabeza | Patas |
| --- | --- | --- | --- | --- |
| ≤22% | fresh ✦ | `>` `<` | sí | anda |
| ≤45% | lively | `>` `<` | sí | anda |
| ≤63% | easy | `o` `o` | sí | anda |
| ≤78% | sluggish | `▬` `▬` | sí | quieto |
| ≤89% | tired | `_` `_` | hundida | quieto |
| <100% | drowning | `x` `x` | hundida | quieto |
| **100%** | k.o. | `x` `x` | hundida | tumbado, patas al aire |

Con **hambre ≥7** los ojos no cambian de forma: se apagan de color. El hambre es
de la otra capa y no toca el estado.

Son **cuatro señales independientes** que se van cayendo en orden: primero los
ojos, luego el paso de las patas, luego la cabeza se hunde y al final la silueta
se tumba. A un vistazo se distingue *tired* de *drowning* sin leer la etiqueta.

**El k.o. ya no necesita puerta trasera.** `StateFor` llevaba un segundo
argumento cuyo único trabajo era forzar el k.o. cuando el contexto llegaba al
100, porque una media de tres números no llega a 100 si no llegan los tres: con
ctx, 5h y 7d al 100, 90 y 90 la media salía 95 —*drowning*— y ese sprite no se
veía nunca. Un número que ya es el contexto llega al 100 él solo.

## Cuando falta el dato

Un CLI viejo no manda `context_window`. Entonces el uso es 0, la mascota sale
fresco y **la banda 1 no dibuja barra**: una barra al 0% sería una medida que no
ha tomado nadie. No se inventa un estado y no se sustituye por una cuota.

## Qué NO mide

Ni el **coste en dólares**, ni el **tiempo de sesión**, ni las **líneas tocadas**,
ni el estado de git, ni la caché, ni —desde esta versión— las **cuotas de la
cuenta**. Todo eso sale en las bandas, pero no le afecta a la mascota.

Tampoco mide el **progreso**. Que la mascota esté *k.o.* no lo devuelve a larva:
la silueta la elige la XP, y la XP no la toca el uso. Un `surgeon` reventado
sigue siendo un surgeon, con cara de haber visto cosas.

Los contadores que abren la rama brasa —`impulsive` a partir de un pico del 85%,
`ctx_maxed` a partir del 95%— leen ese mismo pico de contexto. Durante un tiempo
leyeron el cuello, con el argumento de que una cuota apretada también es trabajar
al límite; el resultado era que abrir cuatro sesiones en paralelo pagaba la rama
sin haber llenado una sola ventana.

## Honestidad

La mascota **no finge emociones**. No se pone contento porque el código compile ni
triste porque falle un test: refleja un número real y comprobable, y ahora
además un número del que la sesión que lo enseña es responsable. Si está
cansado, es que tu ventana va por el 85%.

---

Ver también el [README](../../README.md) para las bandas, la paleta y el resto de la
statusline, y [evolution.md](evolution.md) para la otra capa: XP, hambre,
comida y las 41 evoluciones.
