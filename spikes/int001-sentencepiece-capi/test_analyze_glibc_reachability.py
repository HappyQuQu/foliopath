import unittest

from analyze_glibc_reachability import direct_matches, parse_needed, parse_undefined_symbols


class AnalyzeGlibcReachabilityTest(unittest.TestCase):
    def test_parses_versioned_undefined_symbols(self):
        output = """
0000 DF *UND* 0000 (GLIBC_2.2.5) __isoc99_sscanf
0000 DF .text 0000 scanf
0000 DF *UND* 0000 ungetwc@GLIBC_2.2.5
"""
        self.assertEqual(parse_undefined_symbols(output), ["__isoc99_sscanf", "ungetwc"])

    def test_maps_exact_affected_surfaces(self):
        matches = direct_matches(
            ["__isoc99_sscanf", "fp_nquery", "not_scanf_like", "ungetwc_unlocked"]
        )
        self.assertEqual(matches["CVE-2026-5450"], ["__isoc99_sscanf"])
        self.assertEqual(matches["CVE-2026-5928"], ["ungetwc_unlocked"])
        self.assertEqual(matches["CVE-2026-5435"], ["fp_nquery"])

    def test_parses_needed_entries(self):
        self.assertEqual(
            parse_needed("  NEEDED libm.so.6\n  NEEDED libc.so.6\n"),
            ["libc.so.6", "libm.so.6"],
        )


if __name__ == "__main__":
    unittest.main()
