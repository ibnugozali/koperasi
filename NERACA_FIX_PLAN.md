# Neraca Balance Sheet Fix Plan

## Problem Analysis
The Neraca (Balance Sheet) currently calculates:
- Total ASET (Assets) = Total Aset Lancar + Total Aset Tetap
- Total KEWAJIBAN & EKUITAS = Total Kewajiban Lancar + Total Ekuitas

The issue is that there's no logic to ensure they balance. In accounting:
- Assets must equal Liabilities + Equity
- If they don't balance, there should be a "Selisih" (Difference) item shown

## Solution
Add automatic balancing logic in the JavaScript `updateNeraca()` function:
1. Calculate the difference: selisih = totalAset - totalKewajibanEkuitas
2. Display the "Selisih" row between Total Assets and Total Kewajiban & Ekuitas
3. If selisih = 0, show "BALANCED" (hijau)
4. If selisih ≠ 0, show the difference value and highlight in red

## Changes Required

### File: templates/ketua/ketua_laporan.html

1. Add "Selisih" row in the HTML table (after Total Assets row)
2. Modify `updateNeraca()` function to:
   - Calculate the difference
   - Update the Selisih display
   - Color code based on balanced/unbalanced state
3. Modify `saveNeraca()` function to save the Selisih data
4. Modify `loadNeracaData()` function to load and display Selisih

## Implementation Details

### 1. Add HTML row for Selisih:
```
html
<tr class="table-danger fw-bold" id="selisihRow" style="display: none;">
    <td colspan="2" class="text-start ps-3">Selisih</td>
    <td class="text-end" id="selisih2024">Rp 0</td>
    <td class="text-end" id="selisih2023">Rp 0</td>
    <td colspan="2" class="text-start ps-3">Selisih</td>
    <td class="text-end" id="selisih2024_right">Rp 0</td>
    <td class="text-end" id="selisih2023_right">Rp 0</td>
</tr>
```

### 2. Update updateNeraca() function:
- Calculate selisih = totalAset - totalKewajibanEkuitas
- Show/hide selisih row based on value
- Update both left and right side selisih values with color coding
