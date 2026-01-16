# gravity-sim

Prosty symulator grawitacyjny N-ciał napisany w Go z użyciem biblioteki Ebiten do wizualizacji.

Cechy:
- Symulacja grawitacji (z opcją "antygrawitacji" dla wybranych ciał).
- Metoda czasowa: semi-implicit (symplectic) Euler.
- Możliwość wczytania gotowych układów z plików JSON w `pkg/assets/`.
- Interaktywne sterowanie (pauza, krok, dodawanie ciał, zmiana masy/promienia, tryb blokowania i antygrawitacji).

Wymagania:
- Go 1.25 lub nowsze
- Biblioteka Ebiten (zdefiniowana w `go.mod`)

Szybki start (PowerShell):

```powershell
# z katalogu projektu
go build -o gravity-sim.exe
# uruchom z przykładową konfiguracją (przykłady znajdują się w pkg/assets)
.\gravity-sim.exe -config pkg/assets/solar.json
```

Dostępne pliki konfiguracyjne:
- `pkg/assets/solar.json` — przykładowy układ słoneczny
- `pkg/assets/3body.json` — przykładowy układ 3-ciał
- `pkg/assets/space.json` — układ testowy

Konfiguracja (krótkie objaśnienie):
- `name` — nazwa środowiska
- `dt` — krok czasowy symulacji (float)
- `bodies` — lista ciał, każde z `mass`, `pos` [x,y], `vel` [x,y], `color` (hex)
- `auto_orbit` — jeżeli true, to prędkości orbitalne dla ciał poza pierwszym zostaną ustawione automatycznie (pierwsze ciało traktowane jest jako centralne)

Jak to działa (technicznie):
- Reprezentacja wektorów 2D: `pkg/physics/body.go` (`Vec2`).
- Ciała: `Body` (masa, pozycja, prędkość, przyspieszenie, promień, kolor, flagi `Locked` i `Anti`).
- Siła grawitacji: `pkg/physics/gravity.go` — oblicza przyspieszenie dla ciała biorąc pod uwagę inne ciała z prostym softeningem (parametr epsilon).
- Integrator: `pkg/physics/integrator.go` — semi-implicit Euler (najpierw aktualizacja prędkości, potem pozycji) zapewniający lepszą stabilność energetyczną niż jawny Euler.

Sterowanie (wybrane skróty klawiszowe):
- P — pauza / wznowienie
- N — jeden krok (gdy pauza)
- H — pokaż/ukryj skróty
- L — toggluje Locked (gdy w trybie dodawania albo dla wybranego ciała)
- V — toggluje Anti (antygrawitacja)
- R / T — powiększ / zmniejsz promień (dla zaznaczonego ciała)
- = / - (lub K / J) — zwiększ / zmniejsz masę

Struktura projektu (ważniejsze pliki):
- `main.go` — UI, obsługa wejścia, renderowanie, zarządzanie symulacją
- `pkg/physics/body.go` — definicje wektorów i struktur ciał oraz podstawowe operacje
- `pkg/physics/gravity.go` — obliczanie przyspieszeń grawitacyjnych
- `pkg/physics/integrator.go` — integrator semi-implicit Euler
- `pkg/simulation/config.go` — wczytywanie konfiguracji z JSON i ustawianie prędkości orbitalnych
- `pkg/simulation/simulator.go` — (logika symulatora, zarządzanie krokami)

Rozszerzanie:
- Dodaj nowe pliki JSON w `pkg/assets/` z konfiguracją ciał.
- Możesz dodać inne metody całkowania w `pkg/physics` (np. RK4) i przełączać w `simulation`.

Problemy i debugowanie:
- Jeżeli ciała "wybuchają" przy kroku dt, zmniejsz `dt` w pliku JSON lub zwiększ `epsilon` w `gravity.go` (softening).

Licencja: brak określonej (dodaj plik LICENSE jeśli chcesz udostępnić projekt publicznie)

---

Jeżeli chcesz, mogę teraz:
- Rozwinąć README o diagramy i przykłady wyników.
- Napisać szczegółową dokumentację API (dla plików w `pkg/`).
- Przejść linię po linii przez `pkg/physics/gravity.go` (jeśli chcesz, mogę to zrobić teraz).
