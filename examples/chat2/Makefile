.PHONY: all build fleets-remove fleets-update

all: build

FLEETS := fleets.json
$(FLEETS):
	echo -e $(BUILD_MSG) "Getting latest $(FLEETS)"
	curl -s https://fleets.status.im/ \
		| jq --indent 4 --sort-keys . \
		> $(FLEETS)

fleets-remove:
	rm -f $(FLEETS)

fleets-update: fleets-remove $(FLEETS)

build: $(FLEETS)
	go build -o build/chat2 .
