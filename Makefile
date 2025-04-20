.PHONY: gen-dot

gen-dot:
	@go run ./example/$(DIR) > example/$(DIR)/diagram.dot
	@dot -T png -o example/$(DIR)/diagram.png example/$(DIR)/diagram.dot
	@echo "Generated $(DIR)/diagram.png"
