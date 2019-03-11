CIRCUITS=$(wildcard *.circ)
PDFS=$(patsubst %.circ,%.pdf,$(CIRCUITS))
PNGS=$(patsubst %.circ,%.png,$(CIRCUITS))

all: $(PDFS) $(PNGS)

clean:
	$(RM) $(PDFS) $(PNGS)

%.pdf: %.eps
	ps2pdf -sPAPERSIZE=a4 $<

%.eps: %.dot
	dot -Tps -Gorientation=land -Gsize="11,7.5" $< > $@

%.png: %.dot
	dot -Tpng $< > $@

%.dot: %.circ
	./circuit $< > $@
