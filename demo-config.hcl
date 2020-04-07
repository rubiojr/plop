mountpoint = "/tmp/plop"
default_volume = "example"

volume "example" {
  passphrase = "correct horse battery stable"
  bucket {
    url = "file:///tmp/plopfs-demo"
  }
}

volume "another" {
  passphrase = "slartibartfast"
  bucket {
    url = "file:///tmp/plopfs-demo"
  }
}
