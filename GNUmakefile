
BLOG := ../blog

all:
	@echo "Targets: apidoc"

apidoc:
	$(BLOG)/blog -site -lib $(BLOG) -draft -t templates/mpcl -o ,apidoc docs/apidoc/
	./apps/garbled/garbled -doc ,apidoc pkg
