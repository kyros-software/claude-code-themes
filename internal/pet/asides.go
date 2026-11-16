package pet

// Asides is the second half of each form's voice: five things it says TO the
// person, in character. The Repertoire is the pet talking about itself and its
// work - "he borrado 40 líneas y nadie las va a echar de menos" - and a pet
// that only ever does that is one you watch, not one that is watching you.
// These are the pulla, the advice, the complaint over the shoulder.
//
// They are written to be said at any moment - the idle line turns them over
// every ten minutes, with nothing having happened - so none of them leans on
// an event: no "ese commit", no "esa suite que acaba de pasar".
//
// A mark and its title are the same character at two ages: the title has
// grown into it, and talks from higher up.
var Asides = map[string][]string{
	"spark": {
		"¿y ese botón qué hace? ¿lo pulsas tú o yo?",
		"te miro teclear y no entiendo nada. sigue.",
		"cuando sea grande quiero ser tu linter",
		"¿siempre pones esa cara cuando falla algo?",
		"¿eso se come? ¿y eso? ¿y tu teclado?",
	},

	"pattern": {
		"¿ya has decidido el orden o vamos a ciegas?",
		"hazlo como ayer. ayer salió bien.",
		"una cosa detrás de otra. tú puedes, creo.",
		"no mezcles dos cambios. te estoy viendo.",
		"primero el índice. luego ya escribes.",
	},
	"probe": {
		"¿seguro que es eso? ¿lo has mirado o lo supones?",
		"pregúntale al log antes que a mí",
		"eso es una hipótesis. ¿dónde está el experimento?",
		"me callo si me enseñas la prueba",
		"¿y si lo rompes a propósito, a ver qué dice?",
	},
	"ember": {
		"¿aún lo estás pensando? dale ya.",
		"súbelo. si arde, al menos da calor.",
		"tanto leer te va a quitar las ganas",
		"yo ya lo habría roto dos veces",
		"no preguntes si puedes. prueba y me cuentas.",
	},

	"refactor": {
		"eso que acabas de pegar ya existía. búscalo.",
		"¿de verdad necesitas esa función o te da pena?",
		"ese nombre no dice nada. ponle uno que hable.",
		"borra sin miedo. git se acuerda por ti.",
		"si lo copias una vez más, te lo cobro.",
	},
	"tidy": {
		"¿vas a dejar eso ahí tirado?",
		"límpiate el contexto antes de entrar",
		"no me llenes la sesión de trastos, haz el favor",
		"ese fichero abierto, ¿lo usas o es decoración?",
		"compacta de vez en cuando. se nota en el ambiente.",
	},
	"bughunter": {
		"no lo toques. primero que caiga en el cepo.",
		"esas huellas son de ayer. y son tuyas.",
		"¿has leído el mensaje entero o solo el principio?",
		"deja el cebo y espera. los bichos vuelven.",
		"no era mala suerte. nunca es mala suerte.",
	},
	"architect": {
		"¿dónde va esto? no, en serio: ¿dónde?",
		"antes de escribir, dime qué habla con qué",
		"esa dependencia la vas a pagar. con intereses.",
		"hazme un dibujo. aunque sea feo.",
		"si no cabe en una frase, no lo construyas aún",
	},
	"sprinter": {
		"a tus marcas. ¿listo? pues ya vas tarde.",
		"eso son dos minutos. te cronometro.",
		"no abras otro fichero. pesa en la carrera.",
		"la meta está ahí. deja de atarte los cordones.",
		"corto y fuera. ya lo pulirás otro día. o no.",
	},
	"marathon": {
		"bebe agua. yo no puedo, pero tú sí.",
		"¿has comido? porque yo llevo la cuenta.",
		"estira las piernas, que esto va para largo",
		"no mires el reloj. no ayuda.",
		"guarda fuerzas, que aún queda la parte fea",
	},
	"feral": {
		"¿para qué pides permiso si ya los tienes todos?",
		"no leas el aviso. los avisos son para otros.",
		"llénalo más. aún cabe algo.",
		"ese rm tenía buena pinta. repítelo.",
		"tú conduce. yo voy sin cinturón.",
	},

	"surgeon": {
		"no toques nada que no vayas a operar",
		"¿eso es un cambio o una carnicería?",
		"lávate las manos antes de abrir ese módulo",
		"una línea. he dicho una. cuenta bien.",
		"cose lo que abras. no me dejes puntos sueltos.",
	},
	"weaver": {
		"no tires de ese hilo si no sabes dónde acaba",
		"¿ves cómo todo se toca? pues cuidado.",
		"si cambias aquí, cambia allí. yo te aviso.",
		"ese nudo lo hiciste tú la semana pasada",
		"dame un patrón y te lo repito en veinte sitios",
	},
	"monk": {
		"suelta ese contexto. no era tuyo.",
		"respira antes de dar a enter",
		"¿de verdad necesitas saberlo todo ahora?",
		"el que mucho abarca, compacta tarde",
		"vacía la taza. y el contexto, ya de paso.",
	},
	"gardener": {
		"ese comentario está seco. arráncalo o riégalo.",
		"¿cuánto hace que no miras tu readme?",
		"lo que no documentes hoy lo vas a pisar mañana",
		"no plantes más hasta que crezca lo que hay",
		"esa función vieja aún da fruta. déjala.",
	},
	"bloodhound": {
		"no me cuentes qué crees. enséñame dónde falla.",
		"tú también estás en la lista de sospechosos",
		"¿quién tocó eso el último? ya. me lo temía.",
		"no arregles nada hasta que yo lo huela",
		"«en mi máquina funciona» no es una coartada",
	},
	"exterminator": {
		"¿solo uno? donde hay uno hay nido.",
		"pásame la suite, que huelo movimiento",
		"no celebres. aún se mueve algo ahí dentro.",
		"tú señala y yo fumigo",
		"un test menos es una puerta abierta. tú verás.",
	},
	"cartographer": {
		"¿sabes dónde estás o vamos a ojo?",
		"eso no estaba en el plan. lo apunto, pero me duele.",
		"tacha la tarea cuando acabes. me da paz.",
		"te quedan tres pasos. no te inventes un cuarto.",
		"si te pierdes, vuelve a la lista. para eso está.",
	},
	"oracle": {
		"veo que vas a preguntarme algo. la respuesta es no.",
		"antes de tocar, piensa. lo dice el humo.",
		"tu próximo error empieza con «será rápido»",
		"los signos son claros: no has leído la doc",
		"consulta primero. arrepiéntete menos.",
	},
	"bolt": {
		"¿ya? pues otra.",
		"menos charla, que se me va la corriente",
		"si tardas más, me duermo. y ronco.",
		"un chispazo y listo. no lo enchufes dos veces.",
		"entra, cambia, sal. no hagas amigos.",
	},
	"sniper": {
		"una herramienta. elige bien.",
		"no dispares dos veces a lo mismo",
		"apunta antes. el fichero no se mueve.",
		"sobra ruido. quita la mitad.",
		"si necesitas tres intentos, no era el blanco",
	},
	"ox": {
		"tú dices «un poco más» y yo tiro. otra vez.",
		"¿cuántas horas van? no me lo digas.",
		"si paras tú, paro yo. no pienso insistir.",
		"llevo el yugo, pero el surco lo eliges tú",
		"mañana seguirá ahí. te lo digo por experiencia.",
	},
	"mole": {
		"¿otra vez este fichero? bien. me gusta.",
		"no salgas todavía. aquí abajo está lo bueno.",
		"tú ves un túnel. yo veo un hogar.",
		"si cavas un poco más a la izquierda, sale",
		"no sé qué hay fuera ni falta que me hace",
	},
	"gremlin": {
		"toca ese ajuste. el que pone «no tocar».",
		"¿y si reiniciamos todo, a ver qué pasa?",
		"no era bypass. era confianza. en mí.",
		"instálalo sin leer. vive un poco.",
		"yo no he sido. esta vez has sido tú.",
	},
	"kraken": {
		"¡compacta, que no me quedan brazos!",
		"¿otro fichero más? ¿dónde quieres que lo meta?",
		"me ahogo y tú abriendo pestañas",
		"suelta algo. lo que sea. ya.",
		"esto no es contexto, es una riada",
	},

	"scalpel": {
		"pídeme un corte y dime dónde. nada más.",
		"tu diff tiene grasa. se la quito si me dejas.",
		"yo solo corto. el pulso lo pones tú.",
		"eso no es refactor, es cirugía estética",
		"corto fino. tú procura no moverte.",
	},
	"loom": {
		"tráeme hilo, que estoy parado",
		"tú pones el dibujo y yo no me equivoco",
		"no me cambies el patrón a mitad de tela",
		"cada cambio tuyo es una pasada. no la tuerzas.",
		"tejo lo que me des. dame algo bueno.",
	},
	"abbot": {
		"hijo mío, cierra esa pestaña",
		"la paciencia también compila. más lento.",
		"en este claustro no se grita al terminal",
		"confiesa: ¿cuánto contexto llevas encima?",
		"ora, labora y haz commits pequeños",
	},
	"forest": {
		"aquí crecemos despacio. tú también puedes.",
		"no tales lo que no has plantado",
		"escucha: tus docs hacen ruido de hojas",
		"bajo cada readme hay raíces. no las pises.",
		"quédate un rato. la sombra es gratis.",
	},
	"wolf": {
		"yo no persigo. espero a que vuelvas al sitio.",
		"hueles a prisa. los bugs también lo huelen.",
		"no aúllo por un warning. tú tampoco deberías.",
		"caza conmigo o aparta",
		"ese fallo cojea. déjalo, ya caerá.",
	},
	"wasp": {
		"no me espantes, que estoy trabajando",
		"pico una vez y no vuelve. ¿a cuál?",
		"tu suite zumba bien. no la toques.",
		"donde tú ves verde yo veo territorio",
		"si lo dejas pasar hoy, mañana hay enjambre",
	},
	"atlas": {
		"llevo tu plan entero a cuestas. no añadas más.",
		"mira el mapa antes de preguntar por dónde",
		"eso que llamas atajo es un barranco",
		"cada tarea en su sitio. yo sostengo, tú avanza.",
		"si se cae un paso, se cae el mundo. sin presión.",
	},
	"sphinx": {
		"responde: ¿qué falla de día y pasa de noche?",
		"no te dejo pasar sin un plan",
		"ya sé cómo acaba esto. no te va a gustar.",
		"pregunta bien y contestaré a medias",
		"el enigma no es el código. eres tú.",
	},
	"storm": {
		"aparta, que paso",
		"tres cambios en lo que tú parpadeas",
		"no me pidas calma. no tengo.",
		"truena cuando tardas. ¿lo notas?",
		"llego, mojo y me voy. tú recoges.",
	},
	"falcon": {
		"desde aquí arriba ya he visto el fallo",
		"un picado. uno. mira y aprende.",
		"no aletees tanto, que espantas al blanco",
		"suéltame cuando sepas a qué",
		"tú das vueltas. yo doy una.",
	},
	"mammoth": {
		"llevo aquí desde el primer commit. literalmente.",
		"piso despacio y no me paro. apúntatelo.",
		"¿cansado tú? yo cargo con los colmillos.",
		"el hielo se derrite. yo sigo. tú decide.",
		"otra hora más. total, ya soy un fósil.",
	},
	"worm": {
		"no levantes la cabeza, que perdemos el sitio",
		"llevo días en este repo y aún queda manzana",
		"despacio y por dentro. así se llega.",
		"¿cambiar de proyecto? qué manía con salir.",
		"tú ves lo mismo. yo veo otra capa.",
	},
	"devil": {
		"firma aquí. es solo un --force.",
		"yo te doy el poder. las consecuencias, tú.",
		"¿confirmación? eso es de cobardes.",
		"ese candado lo quito yo con una uña",
		"peca ahora. el rollback ya lo verás mañana.",
	},
	"leviathan": {
		"has despertado algo muy grande. y tiene sueño.",
		"tu contexto ya no cabe en el mar",
		"sigue echando. me lo trago todo. hasta que no.",
		"cuando me dé la vuelta, agárrate",
		"ni tú sabes ya qué llevo dentro",
	},

	"phoenix": {
		"me dejaste morir de hambre. no lo olvido.",
		"ardí, volví y sigues con el mismo bug",
		"tranquilo. si lo rompes, renazco. tú no sé.",
		"otra vez desde las cenizas. ¿a que cansa?",
		"dame de comer antes de que huela a quemado",
	},
	"chimera": {
		"una cabeza dice commit. la otra, test. decide tú.",
		"no nos ponemos de acuerdo. como tú con tu plan.",
		"soy dos mitades. ninguna te da la razón.",
		"pregúntale a la otra cabeza. yo paso.",
		"ruge una y bosteza la otra. así vamos.",
	},
}
