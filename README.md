# cellgen

Générateur d'images (PNG) et d'animations (GIF) à partir d'automates cellulaires 2D.
Go 1.22+, stdlib uniquement.

```
go build -o cellgen .
```

## Exemples

Game of Life, 300 générations en GIF animé :

```
./cellgen -rule B3/S23 -w 200 -h 200 -scale 4 -gens 300 -seed 42 -o life.gif
```

Image fixe de la 200ᵉ génération de HighLife :

```
./cellgen -rule highlife -density 0.35 -gens 200 -o highlife.png
```

Day & Night, animation plus lente (10/100 s par image) :

```
./cellgen -rule daynight -density 0.5 -w 120 -h 120 -scale 3 -gens 150 -delay 10 -o daynight.gif
```

Labyrinthe, avec des bords morts au lieu d'une grille torique :

```
./cellgen -rule maze -density 0.1 -w 120 -h 120 -gens 300 -wrap=false -o maze.png
```

Explosion de Seeds depuis quelques cellules, en noir et blanc :

```
./cellgen -rule seeds -density 0.02 -gens 60 -palette bw -o seeds.gif
```

N'importe quelle règle au format B/S (notation Golly) fonctionne, pas seulement les presets :

```
./cellgen -rule B35678/S5678 -density 0.5 -gens 100 -o custom.png
```

Lister les règles nommées :

```
./cellgen -list-rules
```

## Options

| Flag | Défaut | Rôle |
|---|---|---|
| `-rule` | `B3/S23` | Règle B/S (`B36/S23`, `b2/s`…) ou nom de preset |
| `-list-rules` | | Affiche les presets et quitte |
| `-w`, `-h` | `200` | Taille de la grille en cellules |
| `-scale` | `4` | Pixels par cellule |
| `-gens` | `100` | Nombre de générations, l'état initial compris |
| `-seed` | `1` | Graine aléatoire : même graine, même résultat |
| `-density` | `0.3` | Proportion initiale de cellules vivantes |
| `-wrap` | `true` | Bords toriques ; `-wrap=false` pour des bords morts |
| `-palette` | `age` | `age` (couleur selon l'âge de la cellule) ou `bw` |
| `-delay` | `5` | Délai entre images du GIF, en 1/100 s |
| `-o` | `out.png` | Fichier de sortie : `.gif` produit une animation, sinon un PNG de la dernière génération |

La palette `age` va du jaune pâle (cellule qui vient de naître) au bleu profond
(cellule vivante depuis longtemps), ce qui fait ressortir les structures stables.

## Tests

```
go test ./...
```
