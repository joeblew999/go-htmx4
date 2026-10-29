#!/bin/sh
# Workers Builds' old trigger command, until `mise run charter:builds-configure` points it at charter's launcher
# (.mise/tasks/charter-builds, written by charter tasks): that needs go-htmx4's own build token (#11). Delete
# this file after that.
exec sh .mise/tasks/charter-builds "$@"
