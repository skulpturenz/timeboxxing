import com.google.protobuf.gradle.id

plugins {
    alias(libs.plugins.kotlinJvm)
    alias(libs.plugins.protobuf)
    `java-library`
}

dependencies {
    api(libs.grpc.kotlin.stub)
    api(libs.grpc.protobuf)
    api(libs.grpc.stub)
    api(libs.kotlinx.coroutinesCore)
    api(libs.protobuf.java)
    api(libs.protobuf.kotlin)
    compileOnly(libs.javax.annotationApi)
}

sourceSets {
    main {
        proto {
            srcDir("../../sidecar/proto")
            include("ama/**/*.proto")
            include("projects/**/*.proto")
            include("settings/**/*.proto")
            include("timesheets/**/*.proto")
            include("usage/**/*.proto")
            exclude("transitions/**/*.proto")
        }
    }
}

kotlin {
    sourceSets {
        main {
            kotlin.srcDir("build/generated/source/proto/main/grpckt")
            kotlin.srcDir("build/generated/source/proto/main/kotlin")
        }
    }
}

protobuf {
    protoc {
        artifact = "com.google.protobuf:protoc:${libs.versions.protobuf.get()}"
    }
    plugins {
        id("grpc") {
            artifact = "io.grpc:protoc-gen-grpc-java:${libs.versions.grpcJava.get()}"
        }
        id("grpckt") {
            artifact = "io.grpc:protoc-gen-grpc-kotlin:${libs.versions.grpcKotlin.get()}:jdk8@jar"
        }
    }
    generateProtoTasks {
        all().configureEach {
            builtins {
                id("kotlin")
            }
            plugins {
                id("grpc")
                id("grpckt")
            }
        }
    }
}
