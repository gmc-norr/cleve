# Changelog

## [0.7.0](https://github.com/gmc-norr/cleve/compare/v0.6.0...v0.7.0) (2025-06-18)


### Features

* ability to update analysis path via API ([#107](https://github.com/gmc-norr/cleve/issues/107)) ([03c3b3e](https://github.com/gmc-norr/cleve/commit/03c3b3e1b94f53f45b141d9acfc436293aa773d8))

## [0.6.0](https://github.com/gmc-norr/cleve/compare/v0.5.0...v0.6.0) (2025-06-12)


### Features

* add function for updating run from CLI ([#101](https://github.com/gmc-norr/cleve/issues/101)) ([0957b55](https://github.com/gmc-norr/cleve/commit/0957b553c8e50af9f701b2b50879925f8b743936))
* add moving and incomplete run states ([#106](https://github.com/gmc-norr/cleve/issues/106)) ([c40c0fe](https://github.com/gmc-norr/cleve/commit/c40c0fe548f3f7812980cd5c44288051d2cd67a8))


### Bug Fixes

* extract software versions from older run parameters ([#103](https://github.com/gmc-norr/cleve/issues/103)) ([69b077f](https://github.com/gmc-norr/cleve/commit/69b077fddec667b0ac7d23de8f9a8d411e4d810e))

## [0.5.0](https://github.com/gmc-norr/cleve/compare/v0.4.3...v0.5.0) (2025-06-03)


### Features

* add handling of gene panels ([#88](https://github.com/gmc-norr/cleve/issues/88)) ([8fc0bce](https://github.com/gmc-norr/cleve/commit/8fc0bced11023474e957e39bb899b865c460e15b))
* add loading indicators to plots ([#95](https://github.com/gmc-norr/cleve/issues/95)) ([066baa4](https://github.com/gmc-norr/cleve/commit/066baa413f0706e02e1fd67e3c7375280b685be9))
* add plot for the index summary ([#91](https://github.com/gmc-norr/cleve/issues/91)) ([ac7b7fb](https://github.com/gmc-norr/cleve/commit/ac7b7fbaccbf92b4455a2f3209e9a058999054f0))
* add sequencer software versions to database ([#97](https://github.com/gmc-norr/cleve/issues/97)) ([489245f](https://github.com/gmc-norr/cleve/commit/489245fd6b81e8a6608b0a837d613c729296d338))
* add zoom and consistent series ordering for scatter plots ([#94](https://github.com/gmc-norr/cleve/issues/94)) ([06995de](https://github.com/gmc-norr/cleve/commit/06995de9a3a4b59918e17b88a72ce4a30165479b))
* move scatter plot on run page ([#96](https://github.com/gmc-norr/cleve/issues/96)) ([50966ff](https://github.com/gmc-norr/cleve/commit/50966ff9e39276a084b6398a11c6ae2e224ab1e2))


### Bug Fixes

* allocate space for plots before loading them ([066baa4](https://github.com/gmc-norr/cleve/commit/066baa413f0706e02e1fd67e3c7375280b685be9))
* better styling of plot controls ([066baa4](https://github.com/gmc-norr/cleve/commit/066baa413f0706e02e1fd67e3c7375280b685be9))
* ignore page title when fetching index plots ([#93](https://github.com/gmc-norr/cleve/issues/93)) ([999dd2b](https://github.com/gmc-norr/cleve/commit/999dd2b08f01699c937dbbddd2d09fdf6a07ae1f))
* prevent the index table to shrink horizontally ([999dd2b](https://github.com/gmc-norr/cleve/commit/999dd2b08f01699c937dbbddd2d09fdf6a07ae1f))
* use monospaced font for index sequences in index table ([999dd2b](https://github.com/gmc-norr/cleve/commit/999dd2b08f01699c937dbbddd2d09fdf6a07ae1f))

## [0.4.3](https://github.com/gmc-norr/cleve/compare/v0.4.2...v0.4.3) (2025-05-05)


### Bug Fixes

* bug where the table pagination would be off-by-one if the number of results was evenly divisible by the page size ([9c7007c](https://github.com/gmc-norr/cleve/commit/9c7007c6aa1fd7fac833c24bbe4fb1cbd5e3ed33))
* improved pagination ([#86](https://github.com/gmc-norr/cleve/issues/86)) ([9c7007c](https://github.com/gmc-norr/cleve/commit/9c7007c6aa1fd7fac833c24bbe4fb1cbd5e3ed33))

## [0.4.2](https://github.com/gmc-norr/cleve/compare/v0.4.1...v0.4.2) (2025-04-28)

This release addresses an issue with the document size in the database queries exceeding the maximum size. Queries have been modified to allow for requests of a large number of runs and associated QC data. Some issues with the response for some API endpoints have also been addressed by returning more appropriate HTTP status codes along with better messages.

Another thing that was done that will improve reliability is to host all external JavaScript locally instead of relying on CDNs.

### Bug Fixes

* better status codes for various server side errors ([#84](https://github.com/gmc-norr/cleve/issues/84)) ([ef3d963](https://github.com/gmc-norr/cleve/commit/ef3d963e11b0f427bc50842f8128f2bf68974314))
* mongo document size issue with large page sizes ([#80](https://github.com/gmc-norr/cleve/issues/80)) ([508310b](https://github.com/gmc-norr/cleve/commit/508310b8faa4093d853564b32279274666643a86))

## [0.4.1](https://github.com/gmc-norr/cleve/compare/v0.4.0...v0.4.1) (2025-04-16)


### Bug Fixes

* bug in adding run QC via API ([#78](https://github.com/gmc-norr/cleve/issues/78)) ([a027bcb](https://github.com/gmc-norr/cleve/commit/a027bcb8cc1bcb7ac11b020e91ba8ecd42df79ca))

## [0.4.0](https://github.com/gmc-norr/cleve/compare/v0.3.0...v0.4.0) (2025-04-10)

This version contains quite a lot of things despite the relatively short list of changes. The representation of runs and run QC has changed fundamentally. While the runs are backwards compatible, the run QC is not. Old data will be presented with a message that the QC data needs to be updated. For this there is functionality added to the CLI through `cleve run update`. There are things that are not quite finished, and there are still some things that need some cleaning, and there are separate issues for these things that will be addressed in coming versions.

### Features

* add interop package ([#60](https://github.com/gmc-norr/cleve/issues/60)) ([4066645](https://github.com/gmc-norr/cleve/commit/4066645943a3231dafa87824f075753097308364))
* add pagination for QC API endpoint ([#47](https://github.com/gmc-norr/cleve/issues/47)) ([033c264](https://github.com/gmc-norr/cleve/commit/033c2648cc768697c0cc3d8d37ba30c3fceb9000))
* more flexible interop handling ([#62](https://github.com/gmc-norr/cleve/issues/62)) ([d39f656](https://github.com/gmc-norr/cleve/commit/d39f656edf57bd8e5e9e781c9832478ad834ab9a))
* update to tailwindcss 4 ([5aaddd5](https://github.com/gmc-norr/cleve/commit/5aaddd573a86e76d161b55094b9992e7843b6f52))


### Bug Fixes

* better handling of errors in table pagination ([#45](https://github.com/gmc-norr/cleve/issues/45)) ([1f44f11](https://github.com/gmc-norr/cleve/commit/1f44f11ce8fd10cd999ab76fc55163d85809c807))
* Bump golang.org/x/crypto from 0.22.0 to 0.31.0 ([#58](https://github.com/gmc-norr/cleve/issues/58)) ([ed73806](https://github.com/gmc-norr/cleve/commit/ed738065124c3b0fba4fce9e7df8d2bb8c538541))
* Bump golang.org/x/net from 0.24.0 to 0.33.0 ([#59](https://github.com/gmc-norr/cleve/issues/59)) ([6a5f727](https://github.com/gmc-norr/cleve/commit/6a5f7275f81a666bd41a1da2800c6be34dbf57ab))
* correct filtering for QC data ([#34](https://github.com/gmc-norr/cleve/issues/34)) ([683054e](https://github.com/gmc-norr/cleve/commit/683054eddfec613ecf8930a1f331801dce52ed3b))
* treat sections without suffix as settings ([#32](https://github.com/gmc-norr/cleve/issues/32)) ([787b1fd](https://github.com/gmc-norr/cleve/commit/787b1fd3f5ded294e3ef158fa848580e1ece8eff))

## [0.3.0](https://github.com/gmc-norr/cleve/compare/v0.2.0...v0.3.0) (2024-10-01)

This release adds a basic sample collection to the cleve database.
This is not yet fully featured, and work on this will continue.

One breaking change is an update to how sample sheets are handled.
When updating a sample sheet, instead of replacing it completely, they are instead merged.
Support for UUIDs has also been added.
If a UUID is identified in the RunDescription field in the header of the sample sheet, then this is used as the main identifier if no run ID is associated with the sample sheet.

### Features

* add a basic sample collection to the database ([#22](https://github.com/gmc-norr/cleve/issues/22)) ([37530eb](https://github.com/gmc-norr/cleve/commit/37530ebb0a7d194fce1ae20c3601cd6bc2217701))
* implement custom page out of bounds error ([2663301](https://github.com/gmc-norr/cleve/commit/2663301139b8998c6ff36805ec90f3707f76160d))
* manage sample sheets with UUIDs ([#28](https://github.com/gmc-norr/cleve/issues/28)) ([0c522d9](https://github.com/gmc-norr/cleve/commit/0c522d9e6af7d0a2c957e466bbbb8247bea06817))

## [0.2.0](https://github.com/gmc-norr/cleve/compare/v0.1.0...v0.2.0) (2024-08-22)


### Features

* add version to http headers and page header ([#15](https://github.com/gmc-norr/cleve/issues/15)) ([42c74de](https://github.com/gmc-norr/cleve/commit/42c74def239886b2b7a9cde54d89e0819f2d90fc))
* better versioning ([#11](https://github.com/gmc-norr/cleve/issues/11)) ([ce7b8a8](https://github.com/gmc-norr/cleve/commit/ce7b8a8046aeb370b252f1a1e77dad73401975bc))


### Bug Fixes

* add embed.go to release-please ([#16](https://github.com/gmc-norr/cleve/issues/16)) ([14cbd25](https://github.com/gmc-norr/cleve/commit/14cbd257811433ef870d650cc0c639a8737370d3))
* fallback version fail due to newline ([#14](https://github.com/gmc-norr/cleve/issues/14)) ([386f198](https://github.com/gmc-norr/cleve/commit/386f198848d3759f6ab7d4c0eaf416868c17924c))
* version fallback ([#13](https://github.com/gmc-norr/cleve/issues/13)) ([432eda9](https://github.com/gmc-norr/cleve/commit/432eda90bc4c4eec03ef4b49f11653b1280ca3a2))

## 0.1.0 (2024-08-21)

This is the first release of Cleve with basic functionality in place. More things to come!

### Miscellaneous Chores

* release 0.1.0 ([5c12dd1](https://github.com/gmc-norr/cleve/commit/5c12dd1e24f29a297a5517f78423a213f2f40791))
