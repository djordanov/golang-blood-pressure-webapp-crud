nix build .#bpo-container
podman load < result
podman tag localhost/bpo-container damianjordanov/djordanov:bpo-container
podman push damianjordanov/djordanov:bpo-container
