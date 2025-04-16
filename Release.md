### Notes

*   **Feature Gates Introduced:** This version introduces a new experimental mechanism called Feature Gates. This allows users to enable or disable specific experimental features before they become generally available. Feature gates can be configured in the `featureGates` map within the configuration file.
*   **VirtualNet Feature Gate:** The first available feature gate is `VirtualNet`, which enables the experimental Virtual Network functionality (currently in Alpha stage).

### Features

*   **Virtual Network (VirtualNet):** Introduce experimental virtual network capabilities (Alpha). This allows creating a TUN device managed by frp, enabling Layer 3 connectivity between different clients within the frp network. Requires root/admin privileges and is currently supported on Linux and macOS. Configuration is done via the `virtualNet` section and the `virtual_net` plugin. Enable with feature gate `VirtualNet`. **Note: As an Alpha feature, configuration details may change in future releases.**