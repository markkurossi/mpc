set datafile separator ","

set terminal png size 1400,900 enhanced
set output "ctr.png"

#set terminal epslatex color size 4in,3in
#set output "ctr.tex"

set title "Circuit Cost Per Byte vs Plaintext Size (Log Scale)"
set xlabel "Plaintext size [B]"
set ylabel "Cost per byte"

set logscale y
#set autoscale fix
#set yrange [1:3000]
set grid xtics ytics mxtics mytics

set key top right

plot "ctr_aes.csv"   using 1:($4/$1) with lines lw 2 title "AES128-CTR", \
     "ctr_chacha20.csv" using 1:($4/$1) with lines lw 2 title "ChaCha20"
