#### Build and run the database container

`docker build -t mongoddit .`

Run it

`docker run --name mongoddit -p 27017:27017 -it mongoddit`

#### Connect with the Mongo shell

Run

`docker network inspect bridge`

Find the container name, note the address, then connect

`mongo --host ${NETWORK_ADDRESS}:27017 -u 'username' -p 'password' --authenticationDatabase 'views'`
