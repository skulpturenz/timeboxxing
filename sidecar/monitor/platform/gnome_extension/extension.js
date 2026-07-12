// Timeboxxing Focus Bridge — GNOME Shell extension (GNOME 45+, ESM).
//
// GNOME/Mutter under Wayland exposes no standard protocol for reading the
// focused window of other applications. This extension runs inside GNOME Shell
// (which can see the focus) and re-exports the focused window's app identity,
// title and PID over a private D-Bus method that the Timeboxxing sidecar polls.
//
// It exports the object on the shell's session-bus connection, which owns the
// well-known name "org.gnome.Shell"; the sidecar calls
//   dest=org.gnome.Shell path=/nz/skulpture/Timeboxxing/FocusedWindow
//   method=nz.skulpture.Timeboxxing.FocusedWindow.Get

import Gio from 'gi://Gio';

const FOCUS_IFACE = `
<node>
  <interface name="nz.skulpture.Timeboxxing.FocusedWindow">
    <method name="Get">
      <arg type="s" direction="out" name="json"/>
    </method>
  </interface>
</node>`;

const OBJECT_PATH = '/nz/skulpture/Timeboxxing/FocusedWindow';

export default class TimeboxxingFocusExtension {
    enable() {
        this._dbus = Gio.DBusExportedObject.wrapJSObject(FOCUS_IFACE, this);
        this._dbus.export(Gio.DBus.session, OBJECT_PATH);
    }

    disable() {
        if (this._dbus) {
            this._dbus.unexport();
            this._dbus = null;
        }
    }

    // Get returns a JSON object describing the focused window, or {} when there
    // is no focused window.
    Get() {
        const win = global.display.focus_window;
        if (!win) {
            return JSON.stringify({});
        }
        let pid = 0;
        try {
            pid = win.get_pid() || 0;
        } catch (_e) {
            pid = 0;
        }
        return JSON.stringify({
            wm_class: win.get_wm_class() || '',
            title: win.get_title() || '',
            pid: pid,
        });
    }
}
