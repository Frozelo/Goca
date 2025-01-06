APP_NAME = goca

PORT ?= 3000
ORIGIN ?= http://dummyjson.com

build:
	go build -o $(APP_NAME) main.go

run: build
	./$(APP_NAME) --port $(PORT) --origin $(ORIGIN)

clean:
	rm -f $(APP_NAME)
