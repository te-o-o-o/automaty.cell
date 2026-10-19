# Notes

Décisions, impasses et pièges, et ce qui est vérifié sur quel OS.
[English](notes.md)

## Vérifié, par OS

| | macOS | Linux | Windows |
|---|---|---|---|
| `go vet` et `go test` | CI (`macos-14`) et à la main | CI (`ubuntu-22.04`) | CI (`windows-latest`) |
| Rendus en CLI (PNG, GIF, ZIP) | PNG et GIF à la main, ZIP dans les tests | pas lancé | pas lancé |
| Page web (`-serve`) | à la main (requêtes et page) | pas lancé | pas lancé |
| Page statique (`-site` + WebAssembly) | à la main dans un navigateur (rendu, RANDOM, Muter, annulation, limites, ZIP, masque, langues) | pas lancée | pas lancée |
| MCP (`-mcp`) | tests seulement | tests seulement | tests seulement |
| Binaires de release | compilés par la CI, pas noté comme lancés | compilés par la CI seulement | compilés par la CI seulement |

Les tests génèrent de vrais PNG, GIF et ZIP et pilotent les handlers web et le
serveur MCP : passer sur les trois OS couvre donc plus que la compilation. Il
reste à lancer les binaires à la main sur Linux et Windows.

## Grille et pas de calcul

- Un `uint8` par cellule, quelle que soit la famille : l'âge pour B/S et Larger
  than Life à deux états (plafonné à 255), l'état pour Generations et les
  automates cycliques, V arrondi entre 1 et 255 pour Gray-Scott. Le rendu, les
  palettes, les masques, les symétries et `-at` ne voient que cet octet.
- `swapping` alterne deux grilles au lieu d'en allouer une par génération : la
  grille qu'un pas renvoie n'est valable que jusqu'à l'appel d'après. `generate`,
  `activity` et la planche de contact s'appuient dessus ; garder une copie
  (`slices.Clone`) pour conserver une génération plus longtemps.
- `-at` construit un nouvel automate par changement, et `convert` réécrit les
  cellules quand les deux ne les codent pas pareil (âges, états, bandes
  cycliques).

## Gray-Scott

- Chaque cellule contient deux flottants (U et V), gardés dans la closure de la
  fonction de pas ; `Cells` ne montre que V. Une cellule modifiée de l'extérieur
  depuis le dernier pas (la grille de départ, un masque, un changement `-at`)
  repart de son octet : vivante avec U = V = 0.5, morte vide.
- **Il faut partir d'îlots.** Depuis des cellules aléatoires simples, quelle que
  soit la densité, les cellules se fondent en un mélange uniforme en quelques
  pas et toute la grille s'éteint d'un coup. Mesuré sur les cinq préréglages,
  trois graines et six densités : les îlots de Perlin ont fait pousser des
  motifs à chaque fois dès la densité 0.1, l'aléatoire simple presque jamais.
  Une règle Gray-Scott part donc d'îlots de Perlin de 30 cellules de large quand
  `-noise` vaut 0.
- La valeur d'amorce compte autant : U = 0.5, V = 0.25 (le carré d'amorce des
  manuels) s'éteignait depuis des îlots pour worms à toute densité, et pour reef
  sous 0.5 ; U = V = 0.5 marche
  pour les cinq.
- Les voisins sont additionnés par paires que les miroirs et les quarts de tour
  ne font qu'échanger : un départ symétrique reste exactement symétrique malgré
  les flottants (`TestGrayScott` le vérifie).
- 10 pas de réaction-diffusion par génération, environ 6× le coût d'une règle.
  Les limites web comptent Gray-Scott 6×, mais pas un passage à Gray-Scott via
  `-at`.
- Avec un masque, les cellules mortes autour servent de source de U : les motifs
  prennent un bord clair le long du masque.

## `-stop`

- Il compare toute la grille octet par octet (une empreinte de 64 bits par
  génération). Un motif B/S qui a l'air figé vieillit encore : il ne s'arrête
  qu'une fois toutes les cellules vivantes à 255, jusqu'à 255 générations plus
  tard. Gray-Scott ne montre que V arrondi : il s'arrête quand ça a l'air
  immobile, même si ça dérive en dessous.
- Une répétition avant le dernier changement `-at` ne compte pas : ce
  changement peut relancer le motif.

## RANDOM et Muter

- `cyclicLively` a été mesuré : les seuils qui gardent les automates cycliques en
  mouvement, par voisinage, rayon et nombre d'états.
- RANDOM tire la famille une seule fois, puis retente le reste : retenter la
  famille aussi favoriserait les automates cycliques, qui passent le test
  d'intérêt le plus souvent.
- Muter pondère les nombres de voisins selon leur fréquence : changer 7 ou 8 ne
  change presque jamais rien.
- Les mutants se ressemblaient trop avec 2 à 4 changements et un seuil de
  différence de 0.3 (`minLook`) : désormais 2 à 6 changements et 0.5, à la même
  vitesse (moins d'une demi-seconde par clic). Les pas de Gray-Scott étaient
  bien trop petits (0.001 à 0.003) : désormais jusqu'à 0.012 sur feed et 0.004
  sur kill, le plus sensible. Passer de B/S à Generations un changement sur six
  rendait presque tous les mutants de Life Generations, car les changements
  s'additionnent : un sur douze.

## Page web et MCP

- Les limites web (`server.go`) empêchent une requête de prendre toute la
  machine : cellules × générations, et pixels de toutes les images d'un GIF
  gardées en mémoire.
- `-mask` est un chemin en CLI seulement : la page web envoie l'image, pour que
  le serveur ne lise jamais ses propres fichiers pour un visiteur.
- La page statique (GitHub Pages) est le même template, rendu une fois par
  langue par `-site` (`index.html`, `fr.html`) ; `?lang=fr` sur `index.html`
  redirige vers `fr.html` et le thème est posé dans le navigateur, donc les
  liens partagés depuis `-serve` y marchent tels quels. Son script remplace
  `fetch` par un Web Worker qui fait tourner `automaty.wasm` (`wasm.go`,
  `worker.js`) et emballe la réponse dans une `Response` : le reste de la page
  ne voit pas la différence.
- Build tags : `server.go`, `mcp.go`, `cli.go` (`main`) et leurs tests sont
  `!js` ; `wasm.go` est `js && wasm`. Ce que les deux builds partagent
  (limites, lecture de la requête, réponses) est dans `web.go`.
- Un rendu wasm ne s'interrompt pas depuis Go : un rendu périmé termine son
  worker. La page compile le module une fois et le passe à chaque nouveau
  worker, donc ça coûte peu.
- Le `.wasm` fait environ 5,9 Mo, 1,6 Mo en gzip (Go 1.27, `-ldflags=-s`).
- Le serveur MCP est écrit à la main (outils seulement). Prendre le SDK Go
  officiel si un jour il faut les prompts, les ressources ou le transport HTTP.

## Images du README

- Elles sont générées à leur taille d'affichage (environ 240 à 300 cellules,
  `-scale=1`), puis passées dans `gifsicle -O3`. Les cellules qui vieillissent
  changent de couleur à chaque génération, donc presque chaque pixel change à
  chaque image : moins de couleurs n'aide presque pas. Blobs, spiral et
  mitosis-reef ne gardent en plus qu'une image sur deux avec un délai doublé
  (`gifsicle -O3 in.gif '#0' '#2' … --delay N`, plus `--colors 32` pour
  mitosis-reef), ce qui les divise par deux ; swirls et spaceship sont restés
  complets après comparaison des deux versions. La capture de la page est un JPEG.
- GitHub ne lit les vidéos que si elles sont envoyées par son éditeur web, pas
  depuis le dépôt : ce sera des GIF.

## Pièges

- **`wasm_exec.js` doit venir du Go qui a construit le `.wasm`** : le copier
  depuis `$(go env GOROOT)/lib/wasm` (Go 1.24+) ou `misc/wasm` (avant), ne
  jamais garder une vieille copie.
- **`go vet` pour wasm** inclut les tests : le test d'un fichier `!js` doit
  porter le tag aussi (`GOOS=js GOARCH=wasm go vet .`).

- **Relancer le serveur** : ne tuer que le processus qui écoute sur le port,
  `kill $(lsof -nP -t -iTCP:8080 -sTCP:LISTEN)`. Un simple `lsof -t -i :8080`
  liste aussi le navigateur qui y est connecté, et le tue.
- **Chemins Windows dans du JSON** : un chemin collé dans une chaîne JSON casse
  sur ses barres obliques inverses ; l'encoder avec `json.Marshal`
  (`mcp_test.go`).
- **Les binaires de release utilisent le dernier Go**, pas le minimum de go.mod :
  macOS refuse les binaires du linker de Go 1.22 (« missing LC_UUID »).
