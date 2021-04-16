FROM quay.io/debezium/connect:1.4.1.Final
ENV KAFKA_CONNECT_DEBEZIUM_DIR $KAFKA_CONNECT_PLUGINS_DIR/debezium-connector-postgres
COPY converter-distro.tar.gz $KAFKA_CONNECT_DEBEZIUM_DIR/apicurio-kafka-connect-converter.tar.gz
RUN cd $KAFKA_CONNECT_DEBEZIUM_DIR && tar -xvf apicurio-kafka-connect-converter.tar.gz