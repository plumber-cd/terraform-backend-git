terraform {
  backend "http" {
    address = "http://localhost:6061/?type=git&repository=git@github.com:plumber-cd/terraform-backend-git-fixture-state.git&ref=master&state=state2.json"
    lock_address = "http://localhost:6061/?type=git&repository=git@github.com:plumber-cd/terraform-backend-git-fixture-state.git&ref=master&state=state2.json"
    unlock_address = "http://localhost:6061/?type=git&repository=git@github.com:plumber-cd/terraform-backend-git-fixture-state.git&ref=master&state=state2.json"
    username = "user"
    password = "1234"
  }
}

resource "null_resource" "fixture" {}
