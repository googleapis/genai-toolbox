load("//tools/build_defs/go:go_test.bzl", "go_test")

go_test(
    name = "cloudsqlpg_test",
    srcs = [
        "cloud_sql_pg_create_instances_test.go",
        "cloud_sql_pg_integration_test.go",
        "cloud_sql_pg_pre_check_major_version_upgrade_test.go",
    ],
    library = ":cloudsqlpg",
)
