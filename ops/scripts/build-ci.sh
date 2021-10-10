# build in 2 steps
function build_images() {
    docker buildx bake -f docker-compose.yml builder l2geth l1_chain --set cache-from=/tmp/.buildx-cache,cache-t0=/tmp/.buildx-cache-new
    docker buildx bake -f docker-compose.yml deployer dtl batch_submitter relayer integration_tests --set cache-from=/tmp/.buildx-cache,cache-t0=/tmp/.buildx-cache-new
}

function build_dependencies() {
    yarn
    yarn build
}

build_images &
build_dependencies &

wait
