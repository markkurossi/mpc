
programs := $(wildcard examples/*.mpcl)
circs := $(patsubst %.mpcl,%.circ,$(programs))
mpclcs := $(patsubst %.mpcl,%.mpclc,$(programs))
bristols := $(patsubst %.mpcl,%.bristol,$(programs))
ssas := $(patsubst %.mpcl,%.ssa,$(programs))
dots := $(patsubst %,%.dot,$(circs) $(mpclcs) $(bristols) $(ssas))
svgs := $(patsubst %.dot,%.svg,$(dots))

all:
	@echo $(svgs)

clean:
	@rm -f $(circs) $(mpclcs) $(bristols) $(ssas) $(dots) $(svgs)
