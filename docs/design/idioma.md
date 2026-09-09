# El idioma

El tema hablaba español, y solo español. No por accidente: los nombres del árbol,
el bocadillo de la mascota y las etiquetas del panel salen del lienzo de diseño
«Tema Terminal Claude CLI», que está escrito en español, y la mitad de las frases
de la mascota son chistes que no sobreviven a una traducción («el bug no era el
código, era el jueves»).

El problema no era ese. Era que el español fuera **la única opción**: un plugin
público, descrito en inglés, con un pie de ventana que nadie fuera de aquí puede
leer.

## Qué se traduce y qué no

Se traduce lo que una persona **lee**. No se traduce nada que un fichero
**guarde**.

| | En español | En inglés |
| --- | --- | --- |
| id en `pet.json` | `bughunter` | `bughunter` |
| lo que lees | `cazabugs` | `bughunter` |
| contador | `repro_before_fix` | `repro_before_fix` |
| lo que lees | `reproducir antes de arreglar` | `reproduced before fixing` |

Esa columna del medio es la razón de que cambiar de idioma no cueste nada: el
`pet.json` es el mismo fichero byte a byte, los hooks mandan los mismos eventos y
una racha de nueve días sigue siendo de nueve días. El idioma es una capa de
lectura, no un formato.

Y es también la razón de que **no haya tabla de nombres en inglés**. Los ids ya
son palabras inglesas — el árbol se escribió así — de modo que `Name(id)` en
inglés devuelve el id y no hay nada que mantener sincronizado. La única tabla que
sí hace falta es la de los contadores, porque sus ids son claves y no palabras:
`sessions_under_40` no se lee.

## Dónde vive cada cosa

- `internal/i18n` — qué idioma se habla y quién lo decidió, más el catálogo de
  todo lo que no es un nombre ni una frase de la mascota: etiquetas del panel,
  mensajes de `setup`, la ayuda.
- `internal/pet/names.go` — los nombres del árbol y de los contadores.
- `internal/pet/speech.go` — la voz de la mascota. `RepertoireEN` **no es una
  traducción**: es el mismo chiste, contado otra vez en inglés. Lo que se
  mantiene entre los dos es la *forma* — tres frases por oficio — porque la
  memoria de «sin repetir» tiene tres de fondo y el reinicio del repertorio
  cuenta con ello.

El catálogo es un `struct` y no un mapa de claves a propósito: un campo mal
escrito no compila, y un idioma añadido más tarde no puede olvidarse la mitad sin
que un test lo diga.

## El ajuste

Por orden, gana el primero que conteste:

1. `--lang xx` en la línea de órdenes — para una sola vez.
2. `CCPET_LANG` — para una sesión de terminal.
3. `lang` en `~/.claude/ccpet.json` — `ccpet lang en` lo escribe.
4. El defecto, que es **español**.

`auto` se guarda como `auto` y se resuelve cada vez contra `LC_ALL`,
`LC_MESSAGES` y `LANG`: quien pide que decida el terminal lo pide para siempre, no
para el idioma que su locale dijera esa tarde. `C` y `POSIX` no nombran ningún
idioma, así que se saltan en vez de contestar.

El defecto es el español y no el locale porque actualizar no debe reescribir un
pie de ventana al que alguien está acostumbrado. Quien quiera lo contrario tiene
`ccpet lang auto`, que es una orden y no una sorpresa.

## Lo que sigue en español

Las descripciones de los comandos de barra (`/pet`, `/feed`, `/pet-statusline`)
viven en `commands/*.md`, que Claude Code lee del disco: son estáticas y el
binario no las puede traducir en tiempo de ejecución. Este README y estos
documentos, también.
