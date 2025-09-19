# variables.nix - Import JSON variables and expose as NixOS attributes
# This file is the bridge between JSON data and NixOS configuration

{ config, pkgs, lib, ... }:

let
  # Import the JSON variables file
  # lib.importJSON is a built-in NixOS function that parses JSON into Nix attributes
  varsJson = lib.importJSON ./variables.json;
in
{
  # Expose the imported JSON as a NixOS option that other modules can access
  # This creates config.immichVariables that contains all our settings
  
  options.immichVariables = lib.mkOption {
    type = lib.types.attrs;
    description = "Immich server configuration variables imported from JSON";
  };
  
  config.immichVariables = varsJson;
}

# How this works:
# 1. lib.importJSON reads variables.json and converts it to Nix attributes
# 2. We define a NixOS option called 'immichVariables' 
# 3. We set that option to contain all the JSON data
# 4. Other modules can access this via config.immichVariables.system.timeZone etc.

# Benefits:
# - JSON is easy to generate and parse from Go
# - NixOS validates the imported data
# - Type-safe access to variables throughout the configuration
# - Clear separation between data (JSON) and logic (Nix)

# This approach eliminates:
# - Go template variables scattered through config files
# - Brittle regex parsing of configuration files
# - Manual string manipulation for rollbacks