set datafile separator ","

set terminal png size 1400,900 enhanced
set output "aead.png"

#set terminal epslatex color size 4in,3in
#set output "aead.tex"

set title "Circuit Cost Per Byte vs Plaintext Size (Log Scale)"
set xlabel "Plaintext size [B]"
set ylabel "Cost per byte"

set logscale y
set grid xtics ytics mxtics mytics

set key top right

plot "aead_chacha20poly1305.csv" \
     using 1:($4/$1) with lines lw 2 title "ChaCha20-Poly1305", \
     "aead_aesgcm.csv" \
     using 1:($4/$1) with lines lw 2 title "AES128-GCM"
