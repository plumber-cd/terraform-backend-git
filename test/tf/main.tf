terraform {
  backend "http" {
    address = "http://localhost:6061/?type=git&repository=git@github.com:plumber-cd/terraform-backend-git-fixture-state.git&ref=master&state=state.json"
    lock_address = "http://localhost:6061/?type=git&repository=git@github.com:plumber-cd/terraform-backend-git-fixture-state.git&ref=master&state=state.json"
    unlock_address = "http://localhost:6061/?type=git&repository=git@github.com:plumber-cd/terraform-backend-git-fixture-state.git&ref=master&state=state.json"
  }
}

resource "null_resource" "fixture" {}
