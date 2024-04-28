
BLOG := ../blog
OUTPUT = ,apidoc

all:
	@echo "Targets: apidoc public"

apidoc:
	$(BLOG)/blog -site -lib $(BLOG) -draft -t templates/mpcl -o $(OUTPUT) docs/apidoc/
	./apps/mpcldoc/mpcldoc -dir $(OUTPUT) pkg apps/garbled/examples

public:
	make apidoc OUTPUT=$(HOME)/work/www/mpcl
