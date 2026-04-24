# Fix: OpenVPN DNS polkit authentication popup

## Step 1: Remove the incorrect rule

```bash
sudo rm /etc/polkit-1/rules.d/50-openvpn-dns.rules
```

## Step 2: Create the correct rule

```bash
sudo tee /etc/polkit-1/rules.d/50-update-systemd-resolved.rules << 'POLKITEOF'
/*
 * Allow OpenVPN client services to update systemd-resolved settings.
 * Added by update-systemd-resolved.
 */

function listToBoolMap(list) {
  var result = {};

  for (var i = 0; i < list.length; i++) {
    var item = list[i];
    result[item] = true;
  }

  return result;
}

const updateSystemdResolved = {
  allowedUsers: listToBoolMap(["nobody"]),

  allowedGroups: ["nobody"],

  allowedSubactions: listToBoolMap([
    "set-dns-servers",
    "set-domains",
    "set-default-route",
    "set-llmnr",
    "set-mdns",
    "set-dns-over-tls",
    "set-dnssec",
    "set-dnssec-negative-trust-anchors",
    "revert"
  ]),

  actionIsAllowed: function(action) {
    if ( !action.id.startsWith("org.freedesktop.resolve1.") ) {
      return false;
    }

    var ns = action.id.split(".");
    var subaction = ns[ns.length - 1];

    return this.allowedSubactions[subaction];
  },

  subjectIsAllowed: function(subject) {
    if ( this.allowedUsers[subject.user] ) {
      return true;
    }

    return this.allowedGroups.some(function(group) {
      subject.isInGroup(group);
    });
  },

  isAllowed: function(action, subject) {
    return this.actionIsAllowed(action) && this.subjectIsAllowed(subject);
  }
};

polkit.addRule(function(action, subject) {
  if ( updateSystemdResolved.isAllowed(action, subject) ) {
    return polkit.Result.YES;
  } else {
    return polkit.Result.NOT_HANDLED;
  }
});
POLKITEOF
```

## Step 3: Verify

Disconnect from VPN — the popup should no longer appear. No restart needed.
