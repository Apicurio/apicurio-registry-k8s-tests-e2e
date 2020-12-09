find manifests -name "*.yaml" | xargs sed -e "s,OLD_OPERATOR_IMAGE,NEW_OPERATOR_IMAGE,g" -i
find manifests -name "*.yaml" | xargs cat | grep NEW_OPERATOR_IMAGE