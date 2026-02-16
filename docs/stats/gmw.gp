set datafile separator ","

set terminal png size 1400,900 enhanced
set output "gmw.png"

#set terminal epslatex color size 4in,3in
#set output "aead.tex"

set title "AES128-GCM: Yao vs. GMW-n (Log Scale)"
set xlabel "Plaintext Size [B]"
set ylabel "Speed [Bps]"

set logscale y
set grid xtics ytics mxtics mytics

set key top right

plot "gmw.csv" \
     using 1:3 with lines lw 2 title "Yao", \
     "" using 1:5 with lines lw 2 title "GMW-3", \
     "" using 1:7 with lines lw 2 title "GMW-4"
