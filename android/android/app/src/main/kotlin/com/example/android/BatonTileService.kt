package com.example.android

import android.service.quicksettings.Tile
import android.service.quicksettings.TileService

class BatonTileService : TileService() {
    override fun onStartListening() {
        super.onStartListening()
        qsTile?.state = Tile.STATE_INACTIVE
        qsTile?.label = "Baton handoff"
        qsTile?.updateTile()
    }

    override fun onClick() {
        super.onClick()
        // Toggle owner of selected headphones. Wired via MethodChannel in a real impl.
        qsTile?.state = if (qsTile?.state == Tile.STATE_ACTIVE) Tile.STATE_INACTIVE else Tile.STATE_ACTIVE
        qsTile?.updateTile()
    }
}
