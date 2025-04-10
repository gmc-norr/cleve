# Changelog

## [0.4.0](https://github.com/gmc-norr/cleve/compare/v0.3.0...v0.4.0) (2025-04-10)


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
