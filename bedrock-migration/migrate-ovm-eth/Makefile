migrate:
	go build -o ./migrate ./cmd/main.go
.PHONY: migrate

run: migrate
	scp ./migrate archi-local:/home/ubuntu/migrate
	ssh archi-local 'chmod +x /home/ubuntu/migrate && sudo /home/ubuntu/migrate /var/lib/docker/volumes/'
	sudo migrate