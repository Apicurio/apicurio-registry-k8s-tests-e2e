ARG MANIFESTS_IMAGE
FROM ${MANIFESTS_IMAGE} AS manifests_image

FROM quay.io/operator-framework/upstream-registry-builder:latest

COPY --from=manifests_image /manifests manifests

RUN /bin/initializer -o ./bundles.db
EXPOSE 50051
ENTRYPOINT ["/bin/registry-server"]
CMD ["--database", "/build/bundles.db"]