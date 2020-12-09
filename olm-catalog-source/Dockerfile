ARG MANIFESTS_IMAGE
FROM ${MANIFESTS_IMAGE} AS manifests_image

FROM quay.io/operator-framework/upstream-registry-builder:v1.12.8

COPY --from=manifests_image /manifests manifests

ADD manifests_replacer.sh /manifests_replacer.sh
RUN chmod +x /manifests_replacer.sh && bash /manifests_replacer.sh

RUN /bin/initializer -o ./bundles.db
EXPOSE 50051
ENTRYPOINT ["/bin/registry-server"]
CMD ["--database", "/build/bundles.db"]