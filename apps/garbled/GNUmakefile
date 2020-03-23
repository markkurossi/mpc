
programs := $(wildcard examples/*.mpcl)
circs := $(patsubst %.mpcl,%.circ,$(programs))
ssas := $(patsubst %.mpcl,%.ssa,$(programs))
dots := $(patsubst %,%.dot,$(circs) $(ssas))
svgs := $(patsubst %.dot,%.svg,$(dots))

all:
	@echo $(svgs)

clean:
	@rm -f $(circs) $(ssas) $(dots) $(svgs)
