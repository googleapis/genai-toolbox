load("//tools/build_defs/go:go_library.bzl", "go_library")
load("//tools/build_defs/go:go_test.bzl", "go_test")

go_library(
    name = "cmd",
    srcs = [
        "options.go",
        "root.go",
    ],
    embedsrcs = ["version.txt"],
)

go_test(
    name = "cmd_test",
    srcs = [
        "options_test.go",
        "root_test.go",
    ],
    library = ":cmd",
)
