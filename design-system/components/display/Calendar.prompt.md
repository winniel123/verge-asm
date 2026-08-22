Month grid for schedule visibility and date picks. Event dots ramp on --chart-1 (volume, never severity); arrows move, Enter selects, PageUp/Down change month. Not a popover picker — DateRangePicker stays typed-first.
```jsx
<Calendar month="2026-08" selected={day} onSelect={setDay} events={{ "2026-08-24": 2 }} label="Scheduled runs" />
```
