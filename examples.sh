#!/bin/sh
# Generates a PNG and a GIF for every preset rule (B/S and Generations) and
# for a few cyclic automata, in demo/examples/.
set -e
cd "$(dirname "$0")"
go build -o cellgen .
out=demo/examples
mkdir -p $out

# PNG: 200x200, last generation. GIF: 100x100, 100 frames.
for rule in $(./cellgen -list-rules | awk '{print $1}'); do
	case $rule in
	anneal | daynight | diamoeba) d=0.5 ;; # symmetric rules die out below 0.5
	*) d=0.4 ;;
	esac
	./cellgen -rule $rule -density $d -gens 150 -seed 42 -w 200 -h 200 -scale 4 -o $out/$rule.png
	./cellgen -rule $rule -density $d -gens 100 -seed 42 -w 100 -h 100 -scale 4 -o $out/$rule.gif
	echo "$rule"
done

# Cyclic: states threshold neighborhood radius
while read -r n t nb r; do
	name=cyclic_${n}_${t}_${nb}_r${r}
	flags="-cyclic -states $n -threshold $t -neighborhood $nb -radius $r -palette viridis"
	./cellgen $flags -gens 400 -w 200 -h 200 -scale 3 -o $out/$name.png
	./cellgen $flags -gens 150 -w 100 -h 100 -scale 4 -o $out/$name.gif
	echo "$name"
done <<CYCLIC
3 3 moore 1
4 3 moore 1
8 5 moore 3
14 1 vonneumann 1
6 2 vonneumann 2
CYCLIC
