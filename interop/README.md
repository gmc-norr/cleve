# Interop parsing library in Go

## Supported versions

The support is entirely based on what data I have been able to get my hands on.
If I see new versions, I will add support for those.

Type                  | Filename(s)                                             | Versions
----------------------|---------------------------------------------------------|----------
Run info              | `RunInfo.xml`                                           | 2, 4, 6
Run parameters        | `RunParameters.xml`                                     | MiSeq, NextSeq 550, NovaSeq X Plus
Tile metrics          | `TileMetricsOut.bin`, `TileMetrics.bin`                 | 2, 3
Extended tile metrics | `ExtendedTileMetricsOut.bin`, `ExtendedTileMetrics.bin` | 1, 3
Quality metrics       | `QMetricsOut.bin`, `QMetrics.bin`                       | 4, 6, 7
Error metrics         | `ErrorMetricsOut.bin`, `ErrorMetrics.bin`               | 3, 6
Index metrics         | `IndexMetricsOut.bin`, ``                               | 1, 2
