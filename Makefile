APP_NAME = goca
OUTPUT_DIR = bin

PORT ?= 3000
ORIGIN ?= http://localhost:8080

build:
	mkdir -p $(OUTPUT_DIR)
	go build -o $(OUTPUT_DIR)/$(APP_NAME) main.go

run: build
	$(OUTPUT_DIR)/$(APP_NAME) --port $(PORT) --origin $(ORIGIN)

clean:
	rm -f $(APP_NAME)
