FROM mongo

ENV MONGO_INITDB_DATABASE="views"
ENV MONGO_INITDB_ROOT_USERNAME="username"
ENV MONGO_INITDB_ROOT_PASSWORD="password"

COPY ./init-mongo.js /docker-entrypoint-initdb.d/init-mongo.js