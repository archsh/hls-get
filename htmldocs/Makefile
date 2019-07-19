#### Makefile to Build htmldocs to Golang binary file ##################################################################
HTMLDOCS = $(shell find webadmin/src/ -type f)

all: package
#	@echo "..."

package: assets_vfsdata.go

assets_vfsdata.go: webapp/
	@echo "Generating $@ ..."
	@go run -tags dev packer/packer.go webapp/

clean:
	@echo "Cleaning ..."
	@rm -f assets_vfsdata.go

dist-clean: clean