rootProject.name = "timeboxxing-installer"
enableFeaturePreview("TYPESAFE_PROJECT_ACCESSORS")

pluginManagement {
    repositories {
        google {
            mavenContent {
                includeGroupAndSubgroups("androidx")
                includeGroupAndSubgroups("com.android")
                includeGroupAndSubgroups("com.google")
            }
        }
        mavenCentral()
        gradlePluginPortal()
    }
}

dependencyResolutionManagement {
    repositories {
        google {
            mavenContent {
                includeGroupAndSubgroups("androidx")
                includeGroupAndSubgroups("com.android")
                includeGroupAndSubgroups("com.google")
            }
        }
        mavenCentral()
    }
    versionCatalogs {
        create("libs") {
            from(files("../app/gradle/libs.versions.toml"))
        }
    }
}

plugins {
    id("org.gradle.toolchains.foojay-resolver-convention") version "1.0.0"
}

include(":data")
include(":domain")
include(":packager")
include(":shared")
include(":sidecarApi")

project(":data").projectDir = file("../app/data")
project(":domain").projectDir = file("../app/domain")
project(":packager").projectDir = file("packager")
project(":shared").projectDir = file("../app/shared")
project(":sidecarApi").projectDir = file("../app/sidecarApi")
