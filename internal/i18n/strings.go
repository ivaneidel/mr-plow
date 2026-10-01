package i18n

// strs maps UI keys to their texts. Spanish uses Rioplatense voseo.
var strs = map[string]Text{
	"safety.safe":     {"Safe", "Seguro"},
	"safety.regen":    {"Regenerable", "Regenerable"},
	"safety.review":   {"Review", "Revisar"},
	"safety.personal": {"Personal", "Personal"},

	"badge.dryrun": {"DRY RUN", "SIMULACRO"},
	"hdr.used":     {"used", "usados"},
	"hdr.free":     {"free", "libres"},
	"hdr.of":       {"of", "de"},
	"hdr.other":    {"other", "otros"},

	"st.scanning":      {"scanning", "escaneando"},
	"st.scanned":       {"scanned", "escaneados"},
	"st.files":         {"files", "archivos"},
	"st.dirs":          {"folders", "carpetas"},
	"st.in":            {"in", "en"},
	"st.easy":          {"easy to reclaim", "fácil de liberar"},
	"st.denied":        {"unreadable (Full Disk Access?)", "sin acceso (¿Acceso total al disco?)"},
	"st.showing_cache": {"showing last scan from", "mostrando el último escaneo de"},

	"ago.now":   {"just now", "recién"},
	"ago.min":   {"%d min ago", "hace %d min"},
	"ago.hour":  {"%dh ago", "hace %dh"},
	"ago.day":   {"%d days ago", "hace %d días"},
	"age.today": {"today", "hoy"},

	"tab.suspects": {"Suspects", "Sospechosos"},
	"tab.explore":  {"Explore", "Explorar"},
	"tab.big":      {"Big files", "Archivos grandes"},

	"sel.selected": {"selected", "elegidos"},
	"sel.review":   {"review & remove", "revisar y borrar"},

	"keys.suspects":    {"↑↓ move · space select · enter expand/explore · o reveal", "↑↓ mover · espacio elegir · enter expandir/explorar · o mostrar"},
	"keys.explore":     {"↑↓ move · enter open · ← back · space select · o reveal", "↑↓ mover · enter abrir · ← volver · espacio elegir · o mostrar"},
	"keys.big":         {"↑↓ move · space select · enter show folder · o reveal", "↑↓ mover · espacio elegir · enter ver carpeta · o mostrar"},
	"keys.global":      {"d dry run · c clear · tab switch · r rescan · L lang · ? help · q quit", "d simulacro · c limpiar · tab cambiar · r re-escanear · L idioma · ? ayuda · q salir"},
	"keys.dryrun":      {"↑↓ move · t trash on/off · x drop · e export script · a apply · esc back", "↑↓ mover · t papelera sí/no · x quitar · e exportar script · a aplicar · esc volver"},
	"keys.dryrun_only": {"dry-run mode: ↑↓ move · t trash · x drop · e export script · esc back", "modo simulacro: ↑↓ mover · t papelera · x quitar · e exportar script · esc volver"},
	"keys.report":      {"enter back · q quit", "enter volver · q salir"},

	"flash.rescan":           {"Rescanning…", "Re-escaneando…"},
	"flash.info_only":        {"Shown for awareness only: manage it from its app.", "Solo informativo: gestionalo desde su app."},
	"flash.nothing_selected": {"Select something first (space).", "Primero elegí algo (espacio)."},
	"flash.no_trash":         {"Only plain removals can go to the Trash.", "Solo los borrados simples pueden ir a la Papelera."},
	"flash.dry_run":          {"Dry-run mode: apply is disabled. Press e to export the script.", "Modo simulacro: aplicar está deshabilitado. Apretá e para exportar el script."},
	"flash.nothing_ready":    {"Nothing mr-plow can run itself. Export the script (e) and run it with sudo.", "Nada que mr-plow pueda correr solo. Exportá el script (e) y correlo con sudo."},
	"flash.export_fail":      {"Could not write the script", "No pude escribir el script"},
	"flash.exported":         {"Plan written to", "Plan escrito en"},
	"flash.revealed":         {"Revealed in the file manager.", "Mostrado en el gestor de archivos."},
	"flash.denied":           {"No access. On macOS: System Settings › Privacy & Security › Full Disk Access › add your terminal.", "Sin acceso. En macOS: Ajustes › Privacidad y seguridad › Acceso total al disco › agregá tu terminal."},

	"sus.empty":      {"Nothing found yet…", "Todavía no encontré nada…"},
	"sus.items":      {"items", "ítems"},
	"sus.group_hint": {"space selects every Safe/Regenerable item here (again to unselect) · enter opens the group", "espacio elige todo lo Seguro/Regenerable de acá (otra vez para quitar) · enter abre el grupo"},
	"sus.locations":  {"locations", "ubicaciones"},
	"sus.info_only":  {"Info only: mr-plow won't touch this.", "Solo información: mr-plow no lo toca."},
	"sus.agg_hint":   {"enter lists each folder · space selects them all", "enter lista cada carpeta · espacio las elige todas"},
	"sus.runs":       {"would run", "correría"},
	"sus.empties":    {"would empty", "vaciaría"},
	"sus.removes":    {"would run", "correría"},
	"sus.cached":     {"size from last scan", "tamaño del último escaneo"},

	"group.dev":    {"Dev artifacts", "Artefactos de desarrollo"},
	"group.caches": {"Caches", "Cachés"},
	"group.vms":    {"VMs & containers", "VMs y contenedores"},
	"group.apps":   {"App data", "Datos de apps"},
	"group.big":    {"Old big files", "Archivos grandes viejos"},
	"group.trash":  {"Trash & Downloads", "Papelera y Descargas"},
	"group.media":  {"Media & personal", "Multimedia y personal"},
	"group.system": {"System-wide (sudo)", "Del sistema (sudo)"},

	"groupdesc.dev":    {"Build outputs and dependencies that your tools recreate on demand.", "Salidas de build y dependencias que tus herramientas recrean."},
	"groupdesc.caches": {"Download and app caches. Deleting costs only a re-download.", "Cachés de descargas y apps. Borrarlas solo cuesta volver a bajar."},
	"groupdesc.vms":    {"Disk images of VMs, emulators and container engines. Often the biggest offenders.", "Discos de VMs, emuladores y motores de contenedores. Suelen ser los peores."},
	"groupdesc.apps":   {"Data apps keep for themselves, including leftovers of uninstalled apps.", "Datos que guardan las apps, incluidos restos de apps desinstaladas."},
	"groupdesc.big":    {"Large disk images, archives and videos nobody touched in over a year.", "Imágenes de disco, comprimidos y videos que nadie tocó en más de un año."},
	"groupdesc.trash":  {"Already-deleted things and old downloads.", "Cosas ya borradas y descargas viejas."},
	"groupdesc.media":  {"Your files. Shown so you know where space goes; only you can decide.", "Tus archivos. Para que veas dónde va el espacio; solo vos decidís."},
	"groupdesc.system": {"Outside your home. mr-plow never elevates itself: these go to a sudo script.", "Fuera de tu home. mr-plow nunca se eleva solo: esto va a un script con sudo."},

	"ex.scanning": {"still scanning", "todavía escaneando"},
	"ex.empty":    {"Empty folder.", "Carpeta vacía."},
	"ex.noaccess": {"no access", "sin acceso"},
	"ex.mount":    {"other volume", "otro volumen"},
	"ex.modified": {"newest change", "último cambio"},
	"ex.hint":     {"Sizes are space on disk (what deleting frees). Hard links count once.", "Los tamaños son espacio en disco (lo que libera borrar). Los hard links cuentan una vez."},

	"big.empty": {"No files over 50 MB found (yet).", "No hay archivos de más de 50 MB (todavía)."},
	"big.hint":  {"The 500 biggest files over 50 MB under the scanned folder.", "Los 500 archivos más grandes de más de 50 MB en la carpeta escaneada."},

	"dr.title":           {"Dry run", "Simulacro"},
	"dr.nothing_yet":     {"nothing has been touched yet", "todavía no se tocó nada"},
	"dr.will_run":        {"will run", "se ejecutarán"},
	"dr.need_sudo":       {"need sudo", "necesitan sudo"},
	"dr.blocked":         {"blocked", "bloqueados"},
	"dr.to_trash":        {"to Trash", "a la Papelera"},
	"dr.native":          {"tool command", "comando de la herramienta"},
	"dr.contents":        {"empty folder", "vaciar carpeta"},
	"dr.blocked_because": {"blocked:", "bloqueado:"},
	"dr.sudo_note":       {"not run by mr-plow; goes to the sudo script", "mr-plow no lo corre; va al script con sudo"},
	"dr.confirm":         {"Apply %d actions and free ≈ %s? Removals are permanent (Trash items aren't). y = yes, anything else = no", "¿Aplicar %d acciones y liberar ≈ %s? Los borrados son permanentes (lo de la Papelera no). y = sí, otra tecla = no"},
	"dr.confirm_root":    {"Running as root. Type yes + enter to apply %d actions (≈ %s):", "Corriendo como root. Escribí yes + enter para aplicar %d acciones (≈ %s):"},

	"ap.title": {"Applying…", "Aplicando…"},

	"rep.title":       {"Done", "Listo"},
	"rep.freed":       {"Freed", "Liberado"},
	"rep.skipped":     {"skipped", "salteados"},
	"rep.failed":      {"failed", "fallidos"},
	"rep.disk":        {"Disk", "Disco"},
	"rep.sudo_saved":  {"Root-only actions saved to", "Acciones que requieren root guardadas en"},
	"rep.sudo_hint":   {"Read it, then run it yourself:", "Leelo y después correlo vos:"},
	"rep.log":         {"Every action is logged in", "Cada acción queda registrada en"},
	"rep.rescan_hint": {"Press r on the main screen to rescan.", "Apretá r en la pantalla principal para re-escanear."},

	"help.title": {"help", "ayuda"},
	"help.body": {
		"1 2 3 / tab     switch tab: Suspects · Explore · Big files\n" +
			"↑↓ j k          move     pgup/pgdown  page     g/G  top/bottom\n" +
			"space           select / unselect (on a group: all Safe+Regenerable, again to undo)\n" +
			"enter → l       expand · open folder        ← h backspace  back / fold\n" +
			"o               reveal in Finder / file manager\n" +
			"d               dry run of the selection: exact commands, nothing touched\n" +
			"  in dry run:   t trash on/off · x drop item · e export .sh · a apply\n" +
			"c               clear selection        r  rescan        L  English/Español\n" +
			"q / ctrl+c      quit",
		"1 2 3 / tab     cambiar pestaña: Sospechosos · Explorar · Archivos grandes\n" +
			"↑↓ j k          mover    pgup/pgdown  página   g/G  inicio/fin\n" +
			"espacio         elegir / quitar (en un grupo: todo lo Seguro+Regenerable, otra vez deshace)\n" +
			"enter → l       expandir · abrir carpeta    ← h backspace  volver / plegar\n" +
			"o               mostrar en Finder / gestor de archivos\n" +
			"d               simulacro de lo elegido: comandos exactos, nada se toca\n" +
			"  en simulacro: t papelera sí/no · x quitar · e exportar .sh · a aplicar\n" +
			"c               limpiar selección      r  re-escanear   L  English/Español\n" +
			"q / ctrl+c      salir",
	},
	"help.safety": {
		"Safety: no network code; never follows symlinks; never deletes /, your home or its top folders, system paths or mount points;\n  re-checks every path (same inode) right before deleting; never uses sudo itself; logs every action.",
		"Seguridad: sin código de red; nunca sigue symlinks; nunca borra /, tu home ni sus carpetas principales, rutas del sistema ni puntos de montaje;\n  re-verifica cada ruta (mismo inodo) justo antes de borrar; nunca usa sudo por su cuenta; registra cada acción.",
	},
	"help.close": {"any key closes this help", "cualquier tecla cierra la ayuda"},
}
