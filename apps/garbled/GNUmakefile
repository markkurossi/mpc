
programs := $(wildcard examples/*.mpcl)
circs := $(patsubst %.mpcl,%.circ,$(programs))
mpclcs := $(patsubst %.mpcl,%.mpclc,$(programs))
bristols := $(patsubst %.mpcl,%.bristol,$(programs))
ssas := $(patsubst %.mpcl,%.ssa,$(programs))
dots := $(patsubst %,%.dot,$(circs) $(mpclcs) $(bristols) $(ssas))
svgs := $(patsubst %.dot,%.svg,$(dots))

generated_dots := $(wildcard examples/*.dot)
generated_svgs := $(patsubst %.dot,%.svg,$(generated_dots))

all: $(generated_svgs)

%.svg: %.dot
	dot -Tsvg $+ > $@

clean:
	@rm -f $(circs) $(mpclcs) $(bristols) $(ssas) $(dots) $(svgs)
