# The release build, committed so it cannot first execute during the release it
# must produce (release-pipeline.md §2.3, #1248). Every pull request builds these
# targets with push=false, because the §4.3 guards do not catch a malformed target.

group "default" {
  targets = ["web", "worker"]
}

target "_common" {
  context    = "."
  dockerfile = "Dockerfile"
  platforms  = ["linux/amd64", "linux/arm64"]

  # Empty here, so a pull-request build stays unstamped and the runtime
  # VERGE_VERSION stays reachable (release-pipeline.md §3). The release job
  # overrides it with ${GITHUB_REF_NAME#v}. A release that forgets to publishes
  # unstamped images, so this arg is named here rather than left implicit (#1248).
  args = {
    VERGE_VERSION = ""
  }

  # Empty here, because all three values are release-time. The release job appends
  # revision, version and created. Named rather than left implicit, so a reader of
  # either file sees the seam, the way `args` above does (release-pipeline.md §2.3,
  # #1529).
  #
  # These are the annotations §2.3 owes for rejecting `docker/metadata-action`. They
  # are set here and never at `imagetools create`: annotating there writes a new
  # index, and `publish` signs the digest this build emits (§2.3, §20 item 8).
  annotations = []

  # BuildKit defaults provenance on at mode=min, so silence would give each image a
  # second provenance predicate and break the six-signature model (§2.3).
  #
  # The long form, because the `provenance = false` shorthand is silently ignored on
  # buildx 0.30.1 and the index still carries two attestation manifests (§2.3, #1248).
  attest = [
    "type=provenance,disabled=true",
    "type=sbom,disabled=true",
  ]

  # No cache backend. `COPY . .` sits above every `go build`, so a release commit
  # busts every build layer, and type=gha would spend the 10 GB Actions budget for a
  # hit on `go mod download` alone (§2.3).
}

# Neither target names a tag. The release job sets the output, the tag set and the
# VERGE_VERSION arg. A tag here would be a second place to change them (§2.3).
target "web" {
  inherits = ["_common"]
  target   = "web"
}

target "worker" {
  inherits = ["_common"]
  target   = "worker"
}
