import org.gradle.api.tasks.Sync
import org.gradle.api.GradleException
import org.jetbrains.compose.desktop.application.dsl.TargetFormat

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
    implementation(libs.kotlinx.serializationJson)
    implementation(libs.sentry)
    implementation(libs.jna)

    implementation(libs.compose.uiToolingPreview)
}

val diagnosticsProductionTaskNames = setOf(
    "packageDmg",
    "packageExe",
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

// Produces the jpackage/MSI installer version — which MUST strictly increase for every release the
// updater offers, because Windows MSI only upgrades when ProductVersion increases. Release versions
// look like `X.Y.Z` (stable) or `X.Y.Z-N` (prerelease counter); folding N into the version is what
// makes consecutive prereleases (e.g. 0.0.1-11 -> 0.0.1-12) actually upgrade on Windows instead of
// being no-ops. This affects only the installer/bundle version — the release tag, asset names and
// the in-app appVersion are unaffected.
//
// Mapping to jpackage's three integer fields (first >= 1; macOS rejects a zero leading version, and
// the MSI build field caps at 65535):
//   major = X + 1
//   minor = Y
//   build = Z*1000 + (N for a prerelease, else 999)
// so a prerelease sorts below the stable of the same patch, and the next patch/minor/major always
// sorts higher: 0.0.1-11 -> 1.0.1011, 0.0.1-12 -> 1.0.1012, 0.0.1 -> 1.0.1999, 0.0.2-0 -> 1.0.2000.
fun normalizeInstallerVersion(raw: String): String {
    val trimmed = raw.trim().substringBefore('+')
    val core = trimmed.substringBefore('-')
    val prereleaseToken = trimmed.substringAfter('-', "")

    val parts = core.split('.')
    if (parts.isEmpty() || parts.size > 3) {
        throw GradleException("Invalid package version '$raw'. Expected one to three integers separated by dots.")
    }
    val numbers = parts.map { part ->
        part.toIntOrNull()?.takeIf { it >= 0 }
            ?: throw GradleException("Invalid package version '$raw'. '$part' is not a non-negative integer.")
    }
    val major = numbers[0]
    val minor = numbers.getOrElse(1) { 0 }
    val patch = numbers.getOrElse(2) { 0 }

    val prerelease = if (prereleaseToken.isEmpty()) {
        999 // a stable release sorts above every prerelease of the same patch
    } else {
        prereleaseToken.toIntOrNull()?.takeIf { it in 0..998 }
            ?: throw GradleException(
                "Invalid prerelease counter in package version '$raw'. Expected an integer in 0..998.",
            )
    }
    if (patch > 64) {
        // build = patch*1000 + prerelease must stay within the MSI 16-bit (0..65535) field.
        throw GradleException("Invalid package version '$raw'. Patch component must be <= 64.")
    }

    return "${major + 1}.$minor.${patch * 1000 + prerelease}"
}

val packageVersionProvider = providers.gradleProperty("timeboxxing.packageVersion")
    .orElse("1.0.0")
    .map(::normalizeInstallerVersion)

// The un-normalized release version (e.g. "0.0.5") the running build reports for update checks.
// Distinct from the jpackage-normalized packageVersion above; falls back to it when unset.
val appVersionProvider = providers.gradleProperty("timeboxxing.appVersion")
    .orElse(providers.gradleProperty("timeboxxing.packageVersion"))
    .map { it.trim() }
    .orElse("0.0.0")

val generatedBuildConfigDir = layout.buildDirectory.dir("generated/timeboxxingBuildConfig/kotlin")
val generateDesktopBuildConfig by tasks.registering {
    inputs.property("diagnosticsEnabled", diagnosticsEnabledProvider)
    inputs.property("javaEnv", javaEnvProvider)
    inputs.property("appVersion", appVersionProvider)
    outputs.dir(generatedBuildConfigDir)
    doLast {
        val outputFile = generatedBuildConfigDir.get()
            .file("com/timeboxxing/app/DesktopBuildConfig.kt")
            .asFile
        outputFile.parentFile.mkdirs()
        outputFile.writeText(
            """
            package com.timeboxxing.app

            internal object DesktopBuildConfig {
                const val DiagnosticsEnabled: Boolean = ${diagnosticsEnabledProvider.get()}
                const val JavaEnv: String = "${javaEnvProvider.get()}"
                const val AppVersion: String = "${appVersionProvider.get()}"
            }
            """.trimIndent() + "\n",
        )
    }
}

kotlin {
    sourceSets {
        main {
            kotlin.srcDir(rootProject.layout.projectDirectory.dir("../app/desktopApp/src/main/kotlin"))
            kotlin.srcDir(generatedBuildConfigDir)
        }
    }
}

tasks.named("compileKotlin") {
    dependsOn(generateDesktopBuildConfig)
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

val sidecarExecutableName = if (System.getProperty("os.name").lowercase().contains("windows")) {
    "timeboxxing-sidecar.exe"
} else {
    "timeboxxing-sidecar"
}

val sidecarDir = rootProject.layout.projectDirectory.dir("../sidecar")
val sidecarOutput = layout.buildDirectory.file("sidecar/$sidecarExecutableName")
val installerResourcesRoot = layout.buildDirectory.dir("generated/installerResources")

val buildSidecar by tasks.registering(Exec::class) {
    workingDir = sidecarDir.asFile
    inputs.dir(sidecarDir)
    outputs.file(sidecarOutput)
    commandLine(resolveGoBinary(), "build", "-tags", "assert", "-o", sidecarOutput.get().asFile.absolutePath, ".")
}

val syncInstallerResources by tasks.registering(Sync::class) {
    dependsOn(buildSidecar)
    into(installerResourcesRoot)
    from(sidecarOutput) {
        into("common/sidecar")
    }
    from(rootProject.layout.projectDirectory.dir("../sidecar/db/sqlite-vector")) {
        into("common/sidecar/sqlite-vector")
    }
}

compose.desktop {
    application {
        mainClass = "com.timeboxxing.app.MainKt"

        nativeDistributions {
            targetFormats(TargetFormat.Dmg, TargetFormat.Exe)
            packageName = "Timeboxxing"
            packageVersion = packageVersionProvider.get()
            description = "Timeboxxing desktop app"
            vendor = "Skulpture"
            appResourcesRootDir.set(installerResourcesRoot)

            // jlink strips the runtime image to a default module set. The updater uses
            // java.net.http.HttpClient (java.net.http), which loads at startup, and reaches GitHub
            // over HTTPS with ECDHE cipher suites (jdk.crypto.ec) — neither is in the default set,
            // so both must be requested explicitly or the app crashes on launch / TLS handshake.
            modules("java.net.http", "jdk.crypto.ec")

            macOS {
                bundleID = "com.skulpture.timeboxxing"
                packageName = "Timeboxxing"
                dockName = "Timeboxxing"
            }

            windows {
                // Without these, jpackage installs the app but creates no way to launch it.
                menuGroup = "Timeboxxing"   // Start Menu entry (--win-menu / --win-menu-group)
                shortcut = true             // Desktop shortcut (--win-shortcut)
                perUserInstall = true       // install into the user profile, no admin elevation
                // Stable across every release so upgrades replace the prior install instead of
                // duplicating it. Never regenerate this value.
                upgradeUuid = "3bbf34bc-ad83-4284-a0f6-67c6c654c2c1"
            }
        }
    }
}

tasks.matching {
    it.name in setOf(
        "createDistributable",
        "runDistributable",
        "packageDistributionForCurrentOS",
        "packageDmg",
        "packageExe",
        "prepareAppResources",
    )
}.configureEach {
    dependsOn(syncInstallerResources)
}

// jpackage names the installer "<packageName>-<packageVersion>.<ext>" with no architecture, so a
// DMG built on Apple Silicon and one built on Intel are indistinguishable by filename. Insert the
// arch before the extension after packaging so the produced installer carries it (e.g.
// "Timeboxxing-1.0.0-aarch64.dmg"). The CI release step renames on top of this; local builds get
// the arch too. Maps the JVM os.arch to the same labels the release workflow uses.
val archLabel = when (val osArch = System.getProperty("os.arch").lowercase()) {
    "aarch64", "arm64" -> "aarch64"
    "amd64", "x86_64", "x64" -> "x86_64"
    else -> osArch
}

fun addArchToInstaller(binariesSubdir: String, extension: String) {
    val dir = layout.buildDirectory.dir("compose/binaries/main/$binariesSubdir").get().asFile
    val installer = dir.listFiles { file -> file.isFile && file.name.endsWith(".$extension") }
        ?.firstOrNull { !it.nameWithoutExtension.endsWith("-$archLabel") }
        ?: return
    val renamed = dir.resolve("${installer.nameWithoutExtension}-$archLabel.$extension")
    if (renamed.exists() && !renamed.delete()) {
        throw GradleException("Failed to remove stale installer $renamed")
    }
    if (!installer.renameTo(renamed)) {
        throw GradleException("Failed to add arch suffix to installer $installer")
    }
    logger.lifecycle("Renamed installer to ${renamed.name}")
}

tasks.matching { it.name == "packageDmg" }.configureEach {
    doLast { addArchToInstaller("dmg", "dmg") }
}

tasks.matching { it.name == "packageExe" }.configureEach {
    doLast { addArchToInstaller("exe", "exe") }
}

// jpackage copies app content into the macOS .app with the executable bit stripped (0644), so the
// bundled Go sidecar can't be spawned and the app hangs on its loading screen. Restore the exec bit
// on the app image after createDistributable (which runs before packageDmg), so it works even when
// launched from the read-only DMG. macOS only; Windows spawns a .exe (no exec bit needed).
val isMacOs = System.getProperty("os.name").lowercase().contains("mac")
val macAppImageSidecar =
    layout.buildDirectory.file("compose/binaries/main/app/Timeboxxing.app/Contents/app/resources/sidecar/timeboxxing-sidecar")
val macAppImageDir = layout.buildDirectory.dir("compose/binaries/main/app/Timeboxxing.app")

fun runCommand(vararg command: String): Int {
    val process = ProcessBuilder(*command).redirectErrorStream(true).start()
    process.inputStream.bufferedReader().forEachLine { logger.lifecycle(it) }
    return process.waitFor()
}

tasks.matching { it.name == "createDistributable" }.configureEach {
    doLast {
        if (!isMacOs) {
            return@doLast
        }

        val sidecar = macAppImageSidecar.get().asFile
        if (!sidecar.isFile) {
            throw GradleException("Bundled sidecar not found at $sidecar")
        }
        if (!sidecar.setExecutable(true, false)) {
            throw GradleException("Failed to set the executable bit on $sidecar")
        }

        // Changing mode bits doesn't alter file content, so the ad-hoc resource seal (content-hash
        // based) should stay valid. Verify, and only re-seal if the chmod actually invalidated it —
        // preserving the entitlements the JVM relies on (allow-jit, disable-library-validation, ...).
        val app = macAppImageDir.get().asFile.absolutePath
        if (runCommand("codesign", "--verify", "--deep", "--strict", app) != 0) {
            logger.lifecycle("Re-sealing ad-hoc signature for $app after setting sidecar exec bit")
            val resealed = runCommand(
                "codesign", "--force", "--sign", "-",
                "--preserve-metadata=entitlements,requirements,flags",
                app,
            )
            if (resealed != 0) {
                throw GradleException("Failed to re-seal ad-hoc signature for $app")
            }
        }
    }
}
