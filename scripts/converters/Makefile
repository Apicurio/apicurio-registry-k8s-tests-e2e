
build:
	docker build -t localhost:5000/apicurio-debezium:latest .

push:
	docker push localhost:5000/apicurio-debezium:latest

test-deploy:
	kubectl create deployment apicurio-debezium --image=localhost:5000/apicurio-debezium:latest