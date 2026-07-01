plugins {
    alias(libs.plugins.composeMultiplatform) apply false
    alias(libs.plugins.composeCompiler) apply false
    alias(libs.plugins.kotlinJvm) apply false
    alias(libs.plugins.kotlinMultiplatform) apply false
    alias(libs.plugins.protobuf) apply false
}

tasks.register("createDistributable") {
    group = "distribution"
    description = "Creates the Timeboxxing application image through the packager project."
    dependsOn(":packager:createDistributable")
}

tasks.register("runDistributable") {
    group = "application"
    description = "Runs the packaged Timeboxxing application image through the packager project."
    dependsOn(":packager:runDistributable")
}

tasks.register("packageDistributionForCurrentOS") {
    group = "distribution"
    description = "Packages Timeboxxing for the current OS through the packager project."
    dependsOn(":packager:packageDistributionForCurrentOS")
}

tasks.register("packageDmg") {
    group = "distribution"
    description = "Creates the macOS Timeboxxing DMG through the packager project."
    dependsOn(":packager:packageDmg")
}

tasks.register("packageExe") {
    group = "distribution"
    description = "Creates the Windows Timeboxxing EXE through the packager project."
    dependsOn(":packager:packageExe")
}
