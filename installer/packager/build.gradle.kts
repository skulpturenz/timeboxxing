import org.gradle.api.tasks.Sync
import org.gradle.api.tasks.bundling.Zip
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
    implementation(libs.handlebars)
    implementation(libs.playwright)

    implementation(libs.compose.uiToolingPreview)
}

val diagnosticsProductionTaskNames = setOf(
    "packageDmg",
    "packageExe",
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

// The packager reuses desktopApp's Kotlin sources (above) but not its module, so its classpath
// resources (e.g. the PDF export template + vendored Tailwind runtime) must be bundled explicitly,
// or they are missing from the installer and the app crashes at startup trying to load them.
sourceSets {
    main {
        resources.srcDir(rootProject.layout.projectDirectory.dir("../app/desktopApp/src/main/resources"))
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
            targetFormats(TargetFormat.Dmg, TargetFormat.Exe, TargetFormat.Deb)
            packageName = "Timeboxxing"
            packageVersion = packageVersionProvider.get()
            description = "Timeboxxing desktop app"
            vendor = "Skulpture"
            appResourcesRootDir.set(installerResourcesRoot)

            // jlink strips the runtime image to a default module set. The updater uses
            // java.net.http.HttpClient (java.net.http), which loads at startup, and reaches GitHub
            // over HTTPS with ECDHE cipher suites (jdk.crypto.ec) — neither is in the default set,
            // so both must be requested explicitly or the app crashes on launch / TLS handshake.
            // jdk.zipfs provides the "jar" filesystem provider Playwright's driver uses to unpack
            // itself; without it PDF export fails with 'ProviderNotFoundException: Provider "jar"'.
            // jdk.unsupported provides sun.misc.Unsafe, which gson (bundled by Playwright) needs to
            // instantiate no-arg-less option types like ViewportSize during a render; without it PDF
            // export fails with 'Unable to create instance of class ...ViewportSize'.
            modules("java.net.http", "jdk.crypto.ec", "jdk.zipfs", "jdk.unsupported")

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

            linux {
                // Without a shortcut/menu entry jpackage installs the app under /opt but adds no
                // launcher, so it can only be started from the shell. These add a .desktop entry.
                packageName = "timeboxxing"       // .deb package name (lowercased per Debian policy)
                menuGroup = "Timeboxxing"
                shortcut = true
                appCategory = "Utility"
                debMaintainer = "engineering@skulpture.nz"
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
        "packageDeb",
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

// Windows in-app updates replace the installed jpackage app image directly (no installer run), so
// we publish the app image as a zip alongside the .exe first-time installer. The updater downloads
// this zip, swaps it over the install directory, and relaunches — sidestepping Windows Installer's
// upgrade machinery entirely. Only meaningful on Windows; macOS delivers its .app via the DMG.
val packageWindowsAppImageZip by tasks.registering(Zip::class) {
    dependsOn("createDistributable")
    // The jpackage app image is build/compose/binaries/main/app/Timeboxxing/ — zip it so the archive
    // has a top-level "Timeboxxing/" folder the updater can move straight into %LOCALAPPDATA%.
    from(layout.buildDirectory.dir("compose/binaries/main/app")) {
        include("Timeboxxing/**")
    }
    destinationDirectory.set(layout.buildDirectory.dir("compose/binaries/main/app-image"))
    archiveFileName.set("Timeboxxing-app-image-$archLabel.zip")
}

// jpackage copies app content into the macOS .app with the executable bit stripped (0644), so the
// bundled Go sidecar can't be spawned and the app hangs on its loading screen. Restore the exec bit
// on the app image after createDistributable (which runs before packageDmg), so it works even when
// launched from the read-only DMG. macOS only; Windows spawns a .exe (no exec bit needed).
val isMacOs = System.getProperty("os.name").lowercase().contains("mac")
val macAppImageSidecar =
    layout.buildDirectory.file("compose/binaries/main/app/Timeboxxing.app/Contents/app/resources/sidecar/timeboxxing-sidecar")
val macAppImageDir = layout.buildDirectory.dir("compose/binaries/main/app/Timeboxxing.app")

val isLinux = System.getProperty("os.name").lowercase().contains("linux")
// jpackage strips the exec bit off the bundled sidecar (same reason the macOS step exists). This
// path is the sidecar inside the createDistributable app-image, used by runDistributable and by
// anyone running the app image directly. NOTE: packageDeb does NOT reuse this app image (it invokes
// jpackage --type deb itself), so the .deb is fixed separately by repacking it below.
val linuxAppImageSidecar =
    layout.buildDirectory.file("compose/binaries/main/app/Timeboxxing/lib/app/resources/sidecar/timeboxxing-sidecar")

fun runCommand(vararg command: String): Int {
    val process = ProcessBuilder(*command).redirectErrorStream(true).start()
    process.inputStream.bufferedReader().forEachLine { logger.lifecycle(it) }
    return process.waitFor()
}

// Note: jpackage already encodes the arch in the .deb filename (e.g.
// timeboxxing_2.0.999_amd64.deb), so no addArchToInstaller hook is wired for deb.

// jpackage --type deb strips the exec bit off the bundled sidecar (it lands as 0644 in the package),
// and the .deb installs into a root-owned /opt where the app cannot chmod it at runtime — so the app
// would fail to spawn the sidecar. It also computes Depends only from ELF NEEDED entries, missing the
// runtime tools the app shells out to (secret-tool from libsecret-tools, for the Linux secret store).
// Repack the produced .deb to (1) restore the exec bit and (2) add those runtime dependencies.
// Requires dpkg-deb on the build host (present on Debian/Ubuntu Linux runners).
val extraDebDepends = listOf("libsecret-tools")
tasks.matching { it.name == "packageDeb" }.configureEach {
    doLast {
        if (!isLinux) {
            return@doLast
        }
        val debDir = layout.buildDirectory.dir("compose/binaries/main/deb").get().asFile
        val deb = debDir.listFiles { f -> f.isFile && f.name.endsWith(".deb") }?.firstOrNull()
            ?: throw GradleException("packageDeb produced no .deb in $debDir")

        val work = layout.buildDirectory.dir("deb-exec-fix").get().asFile
        work.deleteRecursively()
        work.mkdirs()

        if (runCommand("dpkg-deb", "-R", deb.absolutePath, work.absolutePath) != 0) {
            throw GradleException("dpkg-deb -R failed for $deb")
        }
        val sidecar = work.resolve("opt/timeboxxing/lib/app/resources/sidecar/timeboxxing-sidecar")
        if (!sidecar.isFile) {
            throw GradleException("Bundled sidecar not found in .deb at $sidecar")
        }
        if (!sidecar.setExecutable(true, false)) {
            throw GradleException("Failed to set the executable bit on $sidecar")
        }

        // Add runtime tool dependencies to the control file's Depends field.
        val control = work.resolve("DEBIAN/control")
        val lines = control.readLines().toMutableList()
        val dependsIndex = lines.indexOfFirst { it.startsWith("Depends:") }
        if (dependsIndex >= 0) {
            val existing = lines[dependsIndex].removePrefix("Depends:").split(",").map { it.trim() }.filter { it.isNotEmpty() }
            val merged = (existing + extraDebDepends.filter { dep -> existing.none { it == dep || it.startsWith("$dep ") } })
            lines[dependsIndex] = "Depends: " + merged.joinToString(", ")
        } else {
            lines.add("Depends: " + extraDebDepends.joinToString(", "))
        }
        control.writeText(lines.joinToString("\n") + "\n")

        if (runCommand("dpkg-deb", "--build", "--root-owner-group", work.absolutePath, deb.absolutePath) != 0) {
            throw GradleException("dpkg-deb --build failed for $deb")
        }
        logger.lifecycle("Repacked ${deb.name} with executable sidecar bit and Depends: ${extraDebDepends.joinToString(", ")}")
    }
}

tasks.matching { it.name == "createDistributable" }.configureEach {
    doLast {
        if (isLinux) {
            val sidecar = linuxAppImageSidecar.get().asFile
            if (!sidecar.isFile) {
                throw GradleException("Bundled sidecar not found at $sidecar")
            }
            if (!sidecar.setExecutable(true, false)) {
                throw GradleException("Failed to set the executable bit on $sidecar")
            }
            return@doLast
        }

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
