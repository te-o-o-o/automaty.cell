# automaty.cell

Générateur d'images (PNG) et d'animations (GIF) à partir d'automates cellulaires 2D.
Go 1.22+, stdlib uniquement.

```
go build -o automaty.cell .
```

## Exemples

Game of Life, 300 générations en GIF animé :

```
./automaty.cell -rule B3/S23 -w 200 -h 200 -scale 4 -gens 300 -seed 42 -o life.gif
```

Image fixe de la 200ᵉ génération de HighLife :

```
./automaty.cell -rule highlife -density 0.35 -gens 200 -o highlife.png
```

Day & Night, animation plus lente (10/100 s par image) :

```
./automaty.cell -rule daynight -density 0.5 -w 120 -h 120 -scale 3 -gens 150 -delay 10 -o daynight.gif
```

Labyrinthe, avec des bords morts au lieu d'une grille torique :

```
./automaty.cell -rule maze -density 0.1 -w 120 -h 120 -gens 300 -wrap=false -o maze.png
```

Explosion de Seeds depuis quelques cellules, en noir et blanc :

```
./automaty.cell -rule seeds -density 0.02 -gens 60 -palette bw -o seeds.gif
```

N'importe quelle règle au format B/S (notation Golly) fonctionne, pas seulement les presets :

```
./automaty.cell -rule B35678/S5678 -density 0.5 -gens 100 -o custom.png
```

Automate Generations (notation `S/B/C` : survie / naissance / nombre d'états) :

```
./automaty.cell -rule 345/2/4 -gens 200 -palette fire -o starwars.gif
./automaty.cell -rule brian -gens 150 -o brian.png
```

Automate cyclique : une cellule à l'état k passe à k+1 (modulo `-states`) si au
moins `-threshold` voisins sont déjà à k+1 :

```
./automaty.cell -cyclic -states 8 -threshold 5 -radius 3 -gens 400 -scale 3 -palette viridis -o spirales.png
./automaty.cell -cyclic -states 14 -threshold 1 -neighborhood vonneumann -gens 500 -scale 3 -palette ocean -o cca.png
./automaty.cell -cyclic -states 3 -threshold 3 -gens 300 -scale 3 -o 313.png
./automaty.cell -cyclic -states 6 -threshold 2 -radius 2 -neighborhood vonneumann -gens 400 -scale 3 -o labyrinthe.png
```

Départ symétrique (mandalas et kaléidoscopes) : le hasard initial est recopié
en miroir, et comme les règles sont symétriques, le motif le reste :

```
./automaty.cell -rule belzhab -symmetry 8 -gens 150 -seed 5 -palette fire -o mandala.png
./automaty.cell -rule starwars -symmetry 8 -gens 120 -seed 9 -palette viridis -o papillon.png
```

Dégradé personnalisé :

```
./automaty.cell -rule belzhab -gens 150 -colors 1a0033,ff3ea5,ffcc00,3ef3ff -o belzhab.png
```

Lister les règles nommées :

```
./automaty.cell -list-rules
```

## Page web

```
./automaty.cell -serve :8080
```

Puis ouvrir http://localhost:8080 : chaque option a son champ, le rendu se
recalcule à chaque changement, et la commande CLI équivalente s'affiche sous
l'image.

- **? RANDOM** tire au hasard une règle (B/S, Generations ou cyclique), une
  palette, une seed, une symétrie et une animation (GIF : vitesse, durée,
  bords), en écartant les automates qui meurent, se figent ou restent du bruit
  (testés sur une petite grille avant).
- **Thème et langue** : sélecteurs en haut à droite. Thème `bonbon` (par
  défaut) ou `arcade`, langue anglais (par défaut) ou français. Aussi par l'URL :
  `?theme=arcade&lang=fr`.
- **Copy link** : l'URL de la page contient les réglages (et le thème, la
  langue), il suffit de l'envoyer pour que l'autre voie la même création.

La page utilise le même moteur que le CLI (même image au pixel près), avec des
limites pour rester utilisable en ligne : grille de 500×500 au plus, échelle 8,
2000 générations, GIF de 200 millions de pixels, 2 rendus à la fois.

## Options

| Flag | Défaut | Rôle |
|---|---|---|
| `-rule` | `B3/S23` | Règle B/S (`B36/S23`, `b2/s`…), Generations S/B/C (`345/2/4`, `/2/3`…) ou nom de preset |
| `-list-rules` | | Affiche les presets et quitte |
| `-w`, `-h` | `380` | Taille de la grille en cellules |
| `-scale` | `2` | Pixels par cellule |
| `-gens` | `100` | Nombre de générations, l'état initial compris |
| `-seed` | `1` | Graine aléatoire : même graine, même résultat |
| `-symmetry` | `1` | Départ en miroir : `1` (aucun), `2`, `4` ou `8` (grille carrée) |
| `-density` | `0.3` | Proportion initiale de cellules vivantes (ignorée en cyclique : états uniformes) |
| `-wrap` | `true` | Bords toriques ; `-wrap=false` pour des bords morts |
| `-palette` | `age` | Dégradé (`-h` les liste toutes) : `age`, `aurora`, `berry`, `bw`, `candy`, `cherry`, `cyber`, `dusk`, `fire`, `forest`, `gold`, `lagoon`, `lavender`, `lime`, `mint`, `mono`, `neon`, `ocean`, `peach`, `rainbow`, `sunset`, `toxic`, `viridis` |
| `-colors` | | Dégradé personnalisé de 2 à 8 couleurs : `1a0033,ff3ea5,ffcc00` (remplace `-palette`) |
| `-palette-from`, `-palette-to` | | Dégradé personnalisé entre deux couleurs `RRGGBB` (raccourci historique de `-colors`) |
| `-cyclic` | `false` | Automate cyclique au lieu de `-rule` |
| `-states` | `14` | Cyclique : nombre d'états (2-256) |
| `-threshold` | `3` | Cyclique : voisins à l'état suivant nécessaires pour avancer |
| `-neighborhood` | `moore` | Cyclique : `moore` (carré) ou `vonneumann` (losange) |
| `-radius` | `1` | Cyclique : rayon du voisinage |
| `-delay` | `5` | Délai entre images du GIF, en 1/100 s |
| `-o` | `out.png` | Fichier de sortie : `.gif` produit une animation, sinon un PNG de la dernière génération |
| `-serve` | | Sert la page web sur cette adresse (`:8080`) au lieu d'écrire un fichier |

Chaque état est placé sur le dégradé :
- B/S : selon l'âge de la cellule (échelle log), de la première couleur (vient de
  naître) à la dernière (vivante depuis longtemps) ; les cellules mortes ont la
  couleur de fond.
- Generations : vivante = première couleur, puis les états mourants jusqu'à la
  dernière ; mortes = fond.
- Cyclique : états répartis régulièrement sur tout le dégradé.

## Tests

```
go test ./...
```
