import org.gradle.api.DefaultTask
import org.gradle.api.GradleException
import org.gradle.api.file.DirectoryProperty
import org.gradle.api.provider.Property
import org.jetbrains.compose.desktop.application.dsl.TargetFormat
import org.gradle.language.jvm.tasks.ProcessResources
import org.gradle.api.tasks.Input
import org.gradle.api.tasks.JavaExec
import org.gradle.api.tasks.OutputDirectory
import org.gradle.api.tasks.TaskAction

abstract class GenerateDesktopBuildConfigTask : DefaultTask() {
    @get:Input
    abstract val diagnosticsEnabled: Property<Boolean>

    @get:Input
    abstract val javaEnv: Property<String>

    @get:OutputDirectory
    abstract val outputDir: DirectoryProperty

    @TaskAction
    fun generate() {
        val outputFile = outputDir.get()
            .file("com/timeboxxing/app/DesktopBuildConfig.kt")
            .asFile
        outputFile.parentFile.mkdirs()
        outputFile.writeText(
            """
            package com.timeboxxing.app

            internal object DesktopBuildConfig {
                const val DiagnosticsEnabled: Boolean = ${diagnosticsEnabled.get()}
                const val JavaEnv: String = "${javaEnv.get()}"
            }
            """.trimIndent() + "\n",
        )
    }
}

plugins {
    alias(libs.plugins.kotlinJvm)
    alias(libs.plugins.composeMultiplatform)
    alias(libs.plugins.composeCompiler)
}

dependencies {
    implementation(projects.shared)
    implementation(projects.data)
    implementation(projects.domain)

    implementation(compose.animation)
    implementation(compose.desktop.currentOs)
    implementation(compose.materialIconsExtended)
    implementation(platform(libs.koin.bom))
    implementation(libs.koin.compose)
    implementation(libs.koin.compose.viewmodel)
    implementation(libs.koin.core)
    implementation(libs.kotlinx.coroutinesSwing)
    implementation(libs.sentry)
    implementation(libs.jna)

    implementation(libs.compose.uiToolingPreview)

    testImplementation(libs.kotlin.testJunit)
    testImplementation(platform(libs.koin.bom))
    testImplementation(libs.koin.test)
}

val sidecarExecutableName = if (System.getProperty("os.name").lowercase().contains("windows")) {
    "timeboxxing-sidecar.exe"
} else {
    "timeboxxing-sidecar"
}

val diagnosticsProductionTaskNames = setOf(
    "packageDmg",
    "packageMsi",
    "packageDeb",
    "packageDistributionForCurrentOS",
    "createDistributable",
    "runDistributable",
)
fun isDiagnosticsProductionTask(taskName: String): Boolean =
    taskName.substringAfterLast(":") in diagnosticsProductionTaskNames

val requestedTaskNames = gradle.startParameter.taskNames.toList()
val defaultDiagnosticsEnabled = requestedTaskNames.none(::isDiagnosticsProductionTask)
val diagnosticsEnabledProvider = providers.gradleProperty("timeboxxing.diagnostics")
    .map { it.toBooleanStrict() }
    .orElse(defaultDiagnosticsEnabled)

val javaEnvProvider = providers.gradleProperty("timeboxxing.javaEnv")
    .orElse(providers.environmentVariable("JAVA_ENV"))
    .map {
        val normalized = it.trim().lowercase()
        if (normalized !in setOf("production", "development", "test", "local")) {
            throw GradleException("Invalid JAVA_ENV '$it'. Supported values: production, development, test, local.")
        }
        normalized
    }
    .orElse("local")

val generatedBuildConfigDir = layout.buildDirectory.dir("generated/timeboxxingBuildConfig/kotlin")
val generateDesktopBuildConfig by tasks.registering(GenerateDesktopBuildConfigTask::class) {
    diagnosticsEnabled.set(diagnosticsEnabledProvider)
    javaEnv.set(javaEnvProvider)
    outputDir.set(generatedBuildConfigDir)
}

kotlin {
    sourceSets {
        main {
            kotlin.srcDir(generatedBuildConfigDir)
        }
    }
}

tasks.named("compileKotlin") {
    dependsOn(generateDesktopBuildConfig)
}

val landingScreenshotSourceSet = sourceSets.create("landingScreenshot") {
    kotlin.srcDir("src/landingScreenshot/kotlin")
    compileClasspath += sourceSets.main.get().output + configurations.named("compileClasspath").get()
    runtimeClasspath += output + compileClasspath + configurations.named("runtimeClasspath").get()
}

tasks.register<JavaExec>("runLandingScreenshot") {
    group = "application"
    description = "Runs a sanitized demo window for capturing landing page screenshots."
    dependsOn(
        tasks.named("compileKotlin"),
        tasks.named(landingScreenshotSourceSet.classesTaskName),
    )
    mainClass.set("com.timeboxxing.app.LandingScreenshotMainKt")
    classpath = landingScreenshotSourceSet.runtimeClasspath
}

fun resolveGoBinary(): String {
    val home = System.getenv("HOME").orEmpty()
    val candidates = listOfNotNull(
        System.getenv("GO_BINARY")?.takeIf { it.isNotBlank() },
        "$home/.local/share/mise/installs/go/1.26.4/bin/go".takeIf { home.isNotBlank() },
        "/opt/homebrew/bin/go",
        "/usr/local/bin/go",
    )
    return candidates.firstOrNull { file(it).exists() } ?: "go"
}

val sidecarOutput = layout.buildDirectory.file("sidecar/$sidecarExecutableName")
val sqliteVectorResources = rootProject.layout.projectDirectory.dir("../sidecar/db/sqlite-vector")
val buildSidecar by tasks.registering(Exec::class) {
    val sidecarDir = rootProject.layout.projectDirectory.dir("../sidecar")
    workingDir = sidecarDir.asFile
    inputs.dir(sidecarDir)
    outputs.file(sidecarOutput)
    commandLine(resolveGoBinary(), "build", "-tags", "assert", "-o", sidecarOutput.get().asFile.absolutePath, ".")
}

tasks.named<ProcessResources>("processResources") {
    dependsOn(buildSidecar)
    from(sidecarOutput) {
        into("sidecar")
    }
    from(sqliteVectorResources) {
        into("sidecar/sqlite-vector")
    }
}

compose.desktop {
    application {
        mainClass = "com.timeboxxing.app.MainKt"

        nativeDistributions {
            targetFormats(TargetFormat.Dmg, TargetFormat.Msi, TargetFormat.Deb)
            packageName = "com.timeboxxing.app"
            packageVersion = "1.0.0"
        }
    }
}
