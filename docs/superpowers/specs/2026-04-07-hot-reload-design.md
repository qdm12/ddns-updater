# Hot-Reload: Config-Änderungen zur Laufzeit übernehmen

## Problem

Nach Config-Änderungen über die WebUI (POST/PUT/DELETE) muss die Applikation neu gestartet werden, damit die neuen Provider-Einträge aktiv werden. Die Config-Datei wird zwar geschrieben, aber die laufenden Provider im Speicher bleiben unverändert.

## Lösung: Database-Level Swap

Nach jeder erfolgreichen Config-Schreiboperation wird die gesamte Config neu geparsed und die In-Memory-Database atomar aktualisiert. Der Update-Loop nimmt Änderungen automatisch mit, da er `db.SelectAll()` bei jeder Iteration aufruft.

Full-Reload statt Einzel-Update, weil ein Config-Eintrag mehrere Provider erzeugen kann (komma-separierte Owners). Bei typisch <50 Einträgen ist Full-Reload einfacher und performant genug.

## Änderungen

### 1. Database: `ReplaceAll` Methode (`internal/data/memory.go`)

Neue Methode, die den gesamten Record-Slice unter Write-Lock austauscht:

```go
func (db *Database) ReplaceAll(newData []records.Record) {
    db.Lock()
    defer db.Unlock()
    db.data = newData
}
```

### 2. Config-Parsing exportieren (`internal/params/json.go`)

`extractAllSettings` ist unexported. Neue exportierte Funktion:

```go
func ParseProviders(configBytes []byte) ([]provider.Provider, []string, error) {
    return extractAllSettings(configBytes)
}
```

### 3. Server Database-Interface erweitern (`internal/server/interfaces.go`)

```go
type Database interface {
    SelectAll() []records.Record
    ReplaceAll(records []records.Record)
}
```

### 4. API-Handler: Reload nach Schreiboperationen (`internal/server/api.go`)

`apiHandlers` bekommt ein `configParser`-Feld (Funktion die Config-Bytes zu Providern parsed).

Neue `reload()` Methode, aufgerufen nach jedem erfolgreichen `writeConfig()`:

1. Config-Datei lesen
2. Parsen → `[]provider.Provider`
3. Bestehende Records via `db.SelectAll()` holen
4. Neue Records erstellen mit `records.New(provider, events)`
5. History matchen: Key = `domain + owner + ipversion` — wenn Match, History/Status/Time übernehmen
6. `db.ReplaceAll(newRecords)` aufrufen

Bei Parse-Fehlern: Config-Datei wird trotzdem geschrieben (war vorher auch so), aber Reload schlägt fehl → HTTP 500 mit Fehlermeldung zurückgeben und vorherige Config-Datei wiederherstellen.

### 5. Handler-Verdrahtung (`internal/server/handler.go`)

`newAPIHandlers` bekommt den Config-Parser übergeben:

```go
func newAPIHandlers(configPath string, db Database, parser ConfigParser) *apiHandlers
```

### 6. Main-Verdrahtung (`cmd/ddns-updater/main.go`)

Parser-Funktion an `newHandler` durchreichen. Kein struktureller Umbau nötig — nur ein zusätzlicher Parameter.

### 7. Frontend: Restart-Banner entfernen (`internal/server/ui/static/app.js`)

Das "Neustart erforderlich"-Banner und zugehörige Logik entfernen, da Änderungen jetzt sofort aktiv werden.

## Datenfluss

```
WebUI POST/PUT/DELETE
  → API Handler
    → writeConfig(config.json)
    → reload():
        → ReadFile(config.json)
        → ParseProviders(bytes) → []Provider
        → SelectAll() → bestehende []Record
        → Match History by domain+owner+ipversion
        → ReplaceAll(neue []Record)
  → Update-Loop: SelectAll() liefert automatisch neue Records
```

## Fehlerbehandlung

- Parse-Fehler nach writeConfig: HTTP 500, Config auf vorherigen Stand zurücksetzen, DB bleibt unverändert
- Einzelner Provider ungültig: gesamter Reload schlägt fehl (konsistent mit Startup-Verhalten)

## Nicht im Scope

- Externer Reload-Trigger (curl, SIGHUP)
- Persistente History-Migration bei Provider-Änderungen
- Hot-Reload bei manueller config.json-Bearbeitung (nur WebUI)
