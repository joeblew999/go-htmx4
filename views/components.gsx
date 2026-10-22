package views

import (
	"time"

	"github.com/gsxhq/gsx"
	"github.com/joeblew999/go-htmx4/intl"
	"github.com/joeblew999/go-htmx4/locales"
	"github.com/joeblew999/go-htmx4/ui"
	"github.com/joeblew999/go-htmx4/ui/icon"
)

// Components is every gsxui component the other pages don't use, each fed by Intl in the page's language: names,
// numbers, dates, sorting, segmenting and plural forms (views/components.go). Each card names its component and says
// what Intl does for it; the interactive ones round-trip through the Worker's Intl over htmx.
component ComponentsPage() {
	<Layout title={locales.ComponentsTitle} description={locales.ComponentsDescription}>
		<section class="flex flex-col gap-3">
			<ui.Breadcrumb>
				<ui.BreadcrumbList>
					<ui.BreadcrumbItem>
						<ui.BreadcrumbLink href={Home{} |> url}>{ locales.NavHome }</ui.BreadcrumbLink>
					</ui.BreadcrumbItem>
					<ui.BreadcrumbSeparator/>
					<ui.BreadcrumbItem>
						<ui.BreadcrumbEllipsis/>
					</ui.BreadcrumbItem>
					<ui.BreadcrumbSeparator/>
					<ui.BreadcrumbItem>
						<ui.BreadcrumbPage>{ locales.ComponentsTitle }</ui.BreadcrumbPage>
					</ui.BreadcrumbItem>
				</ui.BreadcrumbList>
			</ui.Breadcrumb>
			<h1 class="text-3xl font-semibold tracking-tight">{ locales.ComponentsTitle }</h1>
			<p class="text-muted-foreground">{ locales.ComponentsIntro }</p>
			<ui.Alert>
				<icon.Languages/>
				<ui.AlertTitle>{ locales.ComponentsAlertTitle }</ui.AlertTitle>
				<ui.AlertDescription>{ locales.ComponentsAlertBody(intl.Current(ctx).Name()) }</ui.AlertDescription>
			</ui.Alert>
		</section>
		<ComponentsNav/>
		<DemoSection id="dates" title={locales.ComponentsDatesTitle} description={locales.ComponentsDatesDescription}>
			<CalendarDemo/>
			<DatePickerDemo/>
			<CarouselDemo/>
			<ClockDemos/>
		</DemoSection>
		<DemoSection id="numbers" title={locales.ComponentsNumbersTitle} description={locales.ComponentsNumbersDescription}>
			<TableDemo/>
			<ChartDemo/>
			<ProgressDemo/>
			<SliderDemo/>
			<PlaygroundDemo/>
			<AmountDemo/>
			<PaginationDemo/>
			<OtpDemo/>
			<ShapeDemos/>
		</DemoSection>
		<DemoSection id="names" title={locales.ComponentsNamesTitle} description={locales.ComponentsNamesDescription}>
			<ComboboxDemo/>
			<SelectDemo/>
			<CommandDemo/>
			<ScrollAreaDemo/>
			<CollapsibleDemo/>
			<AccordionDemo/>
			<ListDemo/>
		</DemoSection>
		<DemoSection id="text" title={locales.ComponentsTextTitle} description={locales.ComponentsTextDescription}>
			<TextareaDemo/>
			<AvatarDemo/>
		</DemoSection>
		<DemoSection id="menus" title={locales.ComponentsMenusTitle} description={locales.ComponentsMenusDescription}>
			<MenubarDemo/>
			<ContextMenuDemo/>
			<OverlayDemos/>
			<LoadingDemo/>
			<KbdDemo/>
			<SidebarDemo/>
		</DemoSection>
	</Layout>
}

// Demo is one component's card: its name, what Intl does for it, then the component.
component Demo(name string, about intl.Msg, wide bool, children gsx.Node) {
	<ui.Card data-size="sm" class={ "min-w-0", "md:col-span-2": wide }>
		<ui.CardHeader>
			<ui.CardTitle>
				<h3><code translate="no" class="text-sm">{ name }</code></h3>
			</ui.CardTitle>
			<ui.CardDescription>{ about }</ui.CardDescription>
		</ui.CardHeader>
		<ui.CardContent class="flex flex-col gap-3">{ children }</ui.CardContent>
	</ui.Card>
}

// DemoSection is a group of demos under its heading.
component DemoSection(id string, title intl.Msg, description intl.Msg, children gsx.Node) {
	<section id={id} class="flex scroll-mt-4 flex-col gap-4">
		<div class="flex flex-col gap-1">
			<h2 class="text-xl font-semibold tracking-tight">{ title }</h2>
			<p class="text-sm text-muted-foreground">{ description }</p>
		</div>
		<div class="grid gap-4 md:grid-cols-2">{ children }</div>
	</section>
}

// ComponentsNav is the page's sections, in gsxui's navigation menu.
component ComponentsNav() {
	<ui.NavigationMenu>
		<ui.NavigationMenuList>
			<ui.NavigationMenuItem>
				<ui.NavigationMenuTrigger>{ locales.ComponentsNavSections }</ui.NavigationMenuTrigger>
				<ui.NavigationMenuContent>
					<ul class="grid w-72 gap-1 p-1 sm:w-96">
						{ for _, s := range componentSections {
							<li>
								<ui.NavigationMenuLink href=f`#@{s.ID}`>
									<div class="text-sm font-medium">{ s.Title }</div>
									<div class="text-sm text-muted-foreground">{ s.Description }</div>
								</ui.NavigationMenuLink>
							</li>
						} }
					</ul>
				</ui.NavigationMenuContent>
			</ui.NavigationMenuItem>
		</ui.NavigationMenuList>
	</ui.NavigationMenu>
}

component CalendarDemo() {
	<Demo name="ui.Calendar" about={locales.ComponentsCalendarAbout} wide={false}>
		<ui.Calendar
			mode="range"
			month={componentsDay}
			from={componentsDay}
			to={componentsDay.AddDate(0, 0, 6)}
			weekStartsOn={intl.WeekStart(ctx)}
			showOutsideDays={true}
			captionLayout="dropdown"
			fromYear={2020}
			toYear={2032}
			locale={ui.CalendarLocale(intl.CalendarOf(ctx))}
			class="self-start rounded-md border"
			data-intl-text
		/>
		<p class="text-sm">
			{ locales.ComponentsCalendarRange(intl.DateRange(componentsDay, componentsDay.AddDate(0, 0, 6), intl.Options("dateStyle", "long"))) }
		</p>
	</Demo>
}

component DatePickerDemo() {
	<Demo name="ui.Popover · ui.Tooltip · ui.HoverCard" about={locales.ComponentsPopoverAbout} wide={false}>
		<ui.Popover>
			<ui.Button variant="outline" data-gsxui-slot-popover-trigger aria-expanded="false" class="self-start">
				<icon.CalendarDays/>
				{ componentsDay |> date("dateStyle", "long") }
			</ui.Button>
			<ui.PopoverContent class="w-auto p-0">
				<ui.Calendar
					mode="single"
					month={componentsDay}
					selected={[]time.Time{componentsDay}}
					weekStartsOn={intl.WeekStart(ctx)}
					locale={ui.CalendarLocale(intl.CalendarOf(ctx))}
					data-intl-text
				/>
			</ui.PopoverContent>
		</ui.Popover>
		<div class="flex flex-wrap items-center gap-3 text-sm">
			<ui.Tooltip>
				<ui.Button variant="ghost" size="sm" data-gsxui-slot-tooltip-trigger>
					{ componentsDay |> date("dateStyle", "short") }
				</ui.Button>
				<ui.TooltipContent>{ componentsDay |> date("dateStyle", "full") }</ui.TooltipContent>
			</ui.Tooltip>
			<ui.HoverCard>
				<ui.HoverCardTrigger>
					<span class="underline decoration-dotted underline-offset-4">{ locales.ComponentsHoverPosted(intl.Since(componentsDay)) }</span>
				</ui.HoverCardTrigger>
				<ui.HoverCardContent class="flex w-72 flex-col gap-1 text-sm">
					<p class="font-medium">{ componentsDay |> date("dateStyle", "full", "timeStyle", "long") }</p>
					<p class="text-muted-foreground">{ intl.TimeZone(ctx) |> zone }</p>
				</ui.HoverCardContent>
			</ui.HoverCard>
		</div>
	</Demo>
}

component CarouselDemo() {
	<Demo name="ui.Carousel" about={locales.ComponentsCarouselAbout} wide={false}>
		<ui.Carousel orientation="" class="mx-12">
			<ui.CarouselContent orientation="">
				{ for _, c := range Calendars {
					<ui.CarouselItem orientation="">
						<ui.Card data-size="sm">
							<ui.CardContent class="flex min-h-28 flex-col items-center justify-center gap-2 text-center">
								<span class="text-sm text-muted-foreground">{ c |> intl("DisplayNames", "of", "type", "calendar") }</span>
								<span class="font-semibold">{ componentsDay |> date("dateStyle", "full", "calendar", c) }</span>
							</ui.CardContent>
						</ui.Card>
					</ui.CarouselItem>
				} }
			</ui.CarouselContent>
			<ui.CarouselPrevious orientation=""/>
			<ui.CarouselNext orientation=""/>
		</ui.Carousel>
	</Demo>
}

component ClockDemos() {
	<Demo name="ui.Sheet · ui.Toggle" about={locales.ComponentsClocksAbout} wide={false}>
		<ui.Sheet>
			<ui.Button
				variant="outline"
				data-gsxui-slot-sheet-trigger
				aria-haspopup="dialog"
				aria-expanded="false"
				class="self-start"
			>
				<icon.Globe/>
				{ locales.ComponentsSheetOpen }
			</ui.Button>
			<ui.SheetContent side="" hideCloseButton={false}>
				<ui.SheetHeader>
					<ui.SheetTitle>{ locales.ComponentsSheetTitle }</ui.SheetTitle>
					<ui.SheetDescription>{ locales.ComponentsSheetDescription }</ui.SheetDescription>
				</ui.SheetHeader>
				<ul class="flex flex-col gap-3 overflow-y-auto px-4 pb-4 text-sm">
					{ for _, z := range Clocks {
						<li class="flex flex-col">
							<span class="font-medium tabular-nums">{ time.Now() |> date("weekday", "short", "hour", "numeric", "minute", "2-digit", "timeZone", z) }</span>
							<span class="text-muted-foreground">{ z |> zone }</span>
						</li>
					} }
				</ul>
			</ui.SheetContent>
		</ui.Sheet>
		<div class="flex flex-wrap items-center gap-3">
			<ui.Toggle
				variant="outline"
				size="sm"
				hx-on:click=js`document.querySelector("#toggle-clock time").dataset.opts = this.getAttribute("aria-pressed") === "true" ? @{intl.Options("timeStyle", "short")} : @{intl.Options("timeStyle", "medium")}`
			>
				<icon.Timer/>
				{ locales.ComponentsToggleSeconds }
			</ui.Toggle>
			<span id="toggle-clock" class="font-medium tabular-nums">{ time.Now() |> date("timeStyle", "short") }</span>
		</div>
	</Demo>
}

component TableDemo() {
	<Demo name="ui.Table" about={locales.ComponentsTableAbout} wide={true}>
		<ui.Table>
			<ui.TableCaption>{ locales.ComponentsTableCaption }</ui.TableCaption>
			<ui.TableHeader>
				<ui.TableRow>
					<ui.TableHead>{ locales.ComponentsTableCountry }</ui.TableHead>
					<ui.TableHead>{ locales.ComponentsTableTime }</ui.TableHead>
					<ui.TableHead>{ locales.ComponentsTableCurrency }</ui.TableHead>
					<ui.TableHead class="text-end">{ locales.ComponentsTablePopulation }</ui.TableHead>
				</ui.TableRow>
			</ui.TableHeader>
			<ui.TableBody>
				{ for _, c := range Countries(ctx) {
					<ui.TableRow>
						<ui.TableCell class="font-medium">
							{ c.Region |> intl("DisplayNames", "of", "type", "region") }
						</ui.TableCell>
						<ui.TableCell class="tabular-nums">
							{ time.Now() |> date("hour", "numeric", "minute", "2-digit", "timeZone", c.Zone) }
						</ui.TableCell>
						<ui.TableCell>{ c.Currency |> intl("DisplayNames", "of", "type", "currency") }</ui.TableCell>
						<ui.TableCell class="text-end tabular-nums">{ c.Population |> number("notation", "compact") }</ui.TableCell>
					</ui.TableRow>
				} }
			</ui.TableBody>
			<ui.TableFooter>
				<ui.TableRow>
					<ui.TableCell colspan="3">{ locales.ComponentsTableTotal }</ui.TableCell>
					<ui.TableCell class="text-end tabular-nums">{ Population() |> number("notation", "compact") }</ui.TableCell>
				</ui.TableRow>
			</ui.TableFooter>
		</ui.Table>
	</Demo>
}

component ChartDemo() {
	<Demo name="ui.Chart" about={locales.ComponentsChartAbout} wide={false}>
		{{ cfg, data := Chart(ctx) }}
		<ui.Chart config={cfg} data-intl-text>
			<ui.BarChart data={data}>
				<ui.ChartCartesianGrid horizontal/>
				<ui.ChartXAxis key="month"/>
				<ui.ChartTooltip/>
				<ui.ChartBar key="visits"/>
				<ui.ChartBar key="posts"/>
				<ui.ChartLegend/>
			</ui.BarChart>
		</ui.Chart>
	</Demo>
}

component ProgressDemo() {
	<Demo name="ui.Progress" about={locales.ComponentsProgressAbout} wide={false}>
		<div class="flex items-center justify-between gap-4 text-sm">
			<span>{ locales.ComponentsProgressLabel(31, 50) }</span>
			<span class="tabular-nums">{ 0.62 |> number("style", "percent") }</span>
		</div>
		<ui.Progress value={62} aria-label={locales.ComponentsProgressLabel(31, 50)}/>
	</Demo>
}

component SliderDemo() {
	<Demo name="ui.Slider" about={locales.ComponentsSliderAbout} wide={false}>
		<ui.Label for="components-slider">{ locales.ComponentsSliderLabel }</ui.Label>
		<ui.Slider
			id="components-slider"
			value={42}
			min={0}
			max={100}
			step={1}
			hx-on:input=js`document.querySelector("#slider-result [data-intl]").dataset.args = JSON.stringify([this.value])`
		/>
		<output id="slider-result" for="components-slider" class="text-sm font-medium">{ 42 |> number("style", "unit", "unit", "kilometer", "unitDisplay", "long") }</output>
	</Demo>
}

component PlaygroundDemo() {
	<Demo name="ui.Radio · ui.Checkbox" about={locales.ComponentsPlaygroundAbout} wide={false}>
		<form
			class="flex flex-col gap-3"
			hx-on:change=js`const f = new FormData(this); this.querySelector("output [data-intl]").dataset.opts = JSON.stringify({style: "currency", currency: "EUR", currencyDisplay: f.get("display"), useGrouping: f.has("grouping"), signDisplay: f.has("sign") ? "always" : "auto"})`
		>
			<fieldset class="flex flex-col gap-2">
				<legend class="mb-1 text-sm font-medium">{ locales.ComponentsPlaygroundDisplay }</legend>
				{ for _, d := range currencyDisplays {
					<div class="flex items-center gap-2">
						<ui.Radio id=f`display-@{d.ID}` name="display" value={d.ID} checked={d.ID == "symbol"}/>
						<ui.Label for=f`display-@{d.ID}`>{ d.Label }</ui.Label>
					</div>
				} }
			</fieldset>
			<div class="flex items-center gap-2">
				<ui.Checkbox id="playground-grouping" name="grouping" checked/>
				<ui.Label for="playground-grouping">{ locales.ComponentsPlaygroundGrouping }</ui.Label>
			</div>
			<div class="flex items-center gap-2">
				<ui.Checkbox id="playground-sign" name="sign"/>
				<ui.Label for="playground-sign">{ locales.ComponentsPlaygroundSign }</ui.Label>
			</div>
			<output class="text-lg font-semibold">{ 1234567.891 |> number("style", "currency", "currency", "EUR") }</output>
		</form>
	</Demo>
}

component AmountDemo() {
	<Demo name="ui.InputGroup" about={locales.ComponentsAmountAbout} wide={false}>
		<ui.Label for="components-amount">{ locales.ComponentsAmountLabel }</ui.Label>
		<ui.InputGroup>
			<ui.InputGroupAddon>
				<ui.InputGroupText translate="no">EUR</ui.InputGroupText>
			</ui.InputGroupAddon>
			<ui.InputGroupInput
				id="components-amount"
				inputmode="decimal"
				autocomplete="off"
				value={1234.5 |> number |> plain}
				hx-on:input=js`const plain = intl.parseNumber(document.documentElement.lang, this.value), out = document.getElementById("amount-result"); out.hidden = !plain; document.getElementById("amount-invalid").hidden = !!plain; if (plain) { out.querySelector("code").textContent = plain; out.querySelector("[data-intl]").dataset.args = JSON.stringify([plain]) }`
			/>
			<ui.InputGroupAddon align="inline-end">
				<ui.InputGroupText>{ "EUR" |> intl("DisplayNames", "of", "type", "currency") }</ui.InputGroupText>
			</ui.InputGroupAddon>
		</ui.InputGroup>
		<dl id="amount-result" class="grid grid-cols-[auto_1fr] gap-x-3 text-sm">
			<dt class="text-muted-foreground">{ locales.FormatsTryParsed }</dt>
			<dd><code dir="ltr">1234.5</code></dd>
			<dt class="text-muted-foreground">{ locales.ComponentsAmountFormatted }</dt>
			<dd class="font-medium">{ 1234.5 |> number("style", "currency", "currency", "EUR") }</dd>
		</dl>
		<p id="amount-invalid" class="text-sm text-destructive" hidden>{ locales.FormatsTryNotNumber }</p>
	</Demo>
}

component PaginationDemo() {
	<Demo name="ui.Pagination" about={locales.ComponentsPaginationAbout} wide={false}>
		<ui.Pagination>
			<ui.PaginationContent>
				<ui.PaginationItem>
					<ui.PaginationPrevious href="#numbers"/>
				</ui.PaginationItem>
				<ui.PaginationItem>
					<ui.PaginationLink href="#numbers" size="sm">{ 1 |> number }</ui.PaginationLink>
				</ui.PaginationItem>
				<ui.PaginationItem>
					<ui.PaginationEllipsis/>
				</ui.PaginationItem>
				<ui.PaginationItem>
					<ui.PaginationLink href="#numbers" isActive size="sm">{ 1234 |> number }</ui.PaginationLink>
				</ui.PaginationItem>
				<ui.PaginationItem>
					<ui.PaginationEllipsis/>
				</ui.PaginationItem>
				<ui.PaginationItem>
					<ui.PaginationLink href="#numbers" size="sm">{ 5000 |> number }</ui.PaginationLink>
				</ui.PaginationItem>
				<ui.PaginationItem>
					<ui.PaginationNext href="#numbers"/>
				</ui.PaginationItem>
			</ui.PaginationContent>
		</ui.Pagination>
		<p class="text-sm text-muted-foreground">{ locales.ComponentsPaginationPage(1234, 5000) }</p>
	</Demo>
}

component OtpDemo() {
	<Demo name="ui.InputOTP" about={locales.ComponentsOtpAbout} wide={false}>
		<ui.Label for="components-otp">{ locales.ComponentsOtpLabel(6) }</ui.Label>
		<ui.InputOTP
			id="components-otp"
			maxlength="6"
			inputmode="numeric"
			autocomplete="one-time-code"
			data-gsxui-input-otp-pattern="[0-9]"
		>
			<ui.InputOTPGroup>
				<ui.InputOTPSlot/>
				<ui.InputOTPSlot/>
				<ui.InputOTPSlot/>
			</ui.InputOTPGroup>
			<ui.InputOTPSeparator/>
			<ui.InputOTPGroup>
				<ui.InputOTPSlot/>
				<ui.InputOTPSlot/>
				<ui.InputOTPSlot/>
			</ui.InputOTPGroup>
		</ui.InputOTP>
		<p class="text-sm text-muted-foreground">
			{ locales.ComponentsOtpExpires(intl.Duration(10*time.Minute, intl.Options("style", "long"))) }
		</p>
	</Demo>
}

component ShapeDemos() {
	<Demo name="ui.Resizable · ui.AspectRatio · ui.Drawer" about={locales.ComponentsShapesAbout} wide={false}>
		<ui.ResizablePanelGroup
			orientation=""
			class="min-h-24 rounded-lg border"
			hx-on:gsxui:change=js`this.querySelectorAll("data[data-intl]").forEach((el, i) => { el.dataset.args = JSON.stringify([event.detail.sizes[i] / 100]) })`
		>
			<ui.ResizablePanel defaultSize="40%">
				<div class="flex h-full items-center justify-center p-4 text-sm tabular-nums">
					{ 0.4 |> number("style", "percent") }
				</div>
			</ui.ResizablePanel>
			<ui.ResizableHandle orientation="" withHandle={true}/>
			<ui.ResizablePanel defaultSize="60%">
				<div class="flex h-full items-center justify-center p-4 text-sm tabular-nums">
					{ 0.6 |> number("style", "percent") }
				</div>
			</ui.ResizablePanel>
		</ui.ResizablePanelGroup>
		<ui.AspectRatio
			ratio="16 / 9"
			class="flex items-center justify-center rounded-lg border bg-muted text-sm text-foreground"
		>
			{ locales.ComponentsAspectLabel(1920, 1080) }
		</ui.AspectRatio>
		<ui.Drawer>
			<ui.Button
				variant="outline"
				data-gsxui-slot-drawer-trigger
				aria-haspopup="dialog"
				aria-expanded="false"
				class="self-start"
			>
				<icon.Ruler/>
				{ locales.ComponentsDrawerOpen }
			</ui.Button>
			<ui.DrawerContent direction="">
				<ui.DrawerHeader>
					<ui.DrawerTitle>{ locales.ComponentsDrawerTitle }</ui.DrawerTitle>
					<ui.DrawerDescription>{ locales.ComponentsDrawerDescription }</ui.DrawerDescription>
				</ui.DrawerHeader>
				<ul class="grid gap-2 px-4 text-sm sm:grid-cols-2">
					{ for _, u := range Units {
						<li>{ u.Amount |> number("style", "unit", "unit", u.Unit, "unitDisplay", "long") }</li>
					} }
				</ul>
				<ui.DrawerFooter>
					<ui.Button variant="outline" data-gsxui-dialog-close>{ locales.HomeDialogClose }</ui.Button>
				</ui.DrawerFooter>
			</ui.DrawerContent>
		</ui.Drawer>
	</Demo>
}

component ComboboxDemo() {
	<Demo name="ui.Combobox" about={locales.ComponentsComboboxAbout} wide={false}>
		<ui.Label for="components-country">{ locales.ComponentsComboboxLabel }</ui.Label>
		<ui.Combobox name="country" value="">
			<ui.ComboboxInput
				id="components-country"
				placeholder={locales.ComponentsComboboxPlaceholder |> text}
				showTrigger
				class="w-full"
			/>
			<ui.ComboboxContent>
				<ui.ComboboxList>
					<ui.ComboboxEmpty>{ locales.ComponentsComboboxEmpty }</ui.ComboboxEmpty>
					{ for _, r := range Sorted(ctx, "region", regionCodes) {
						<ui.ComboboxItem value={r.Code} selected={false}>
							<span data-intl-text>{ r.Name }</span>
						</ui.ComboboxItem>
					} }
				</ui.ComboboxList>
			</ui.ComboboxContent>
		</ui.Combobox>
	</Demo>
}

component SelectDemo() {
	<Demo name="ui.Select" about={locales.ComponentsSelectAbout} wide={false}>
		<ui.Select
			name="calendar"
			hx-on:change=js`document.querySelector("#calendar-result [data-intl]").dataset.opts = JSON.stringify({dateStyle: "full", calendar: event.target.value})`
		>
			<ui.SelectTrigger class="w-full" aria-label={locales.ComponentsSelectLabel}>
				<ui.SelectValue placeholder={locales.ComponentsSelectLabel |> text}/>
			</ui.SelectTrigger>
			<ui.SelectContent>
				<ui.SelectGroup>
					{ for _, c := range Calendars {
						<ui.SelectItem value={c} selected={c == "gregory"} disabled={false}>
							{ c |> intl("DisplayNames", "of", "type", "calendar") }
						</ui.SelectItem>
					} }
				</ui.SelectGroup>
			</ui.SelectContent>
		</ui.Select>
		<output id="calendar-result" class="font-medium">{ componentsDay |> date("dateStyle", "full", "calendar", "gregory") }</output>
	</Demo>
}

component CommandDemo() {
	<Demo name="ui.Command · ui.DropdownMenu" about={locales.ComponentsCommandAbout} wide={false}>
		<ui.Command class="rounded-lg border">
			<ui.CommandInput placeholder={locales.ComponentsCommandPlaceholder |> text}/>
			<ui.CommandList class="max-h-56">
				<ui.CommandEmpty>{ locales.ComponentsCommandEmpty }</ui.CommandEmpty>
				<ui.CommandGroup heading={locales.ComponentsCommandLanguages |> text}>
					{ for _, l := range intl.Links(ctx) {
						<ui.CommandItem value={l.Lang} data-href={l.Href}>
							<icon.Languages/>
							<span lang={l.Lang} translate="no">{ l.Name }</span>
						</ui.CommandItem>
					} }
				</ui.CommandGroup>
				<ui.CommandSeparator/>
				<ui.CommandGroup heading={locales.ComponentsCommandPages |> text}>
					<ui.CommandItem value="board" data-href={Board{} |> url}>
						<icon.LayoutGrid/>
						<span>{ locales.NavBoard }</span>
					</ui.CommandItem>
					<ui.CommandItem value="formats" data-href={Formats{} |> url}>
						<icon.Globe/>
						<span>{ locales.NavFormats }</span>
					</ui.CommandItem>
					<ui.CommandItem value="about" data-href={About{} |> url}>
						<icon.Info/>
						<span>{ locales.NavAbout }</span>
					</ui.CommandItem>
				</ui.CommandGroup>
			</ui.CommandList>
		</ui.Command>
		<ui.DropdownMenu>
			<ui.Button
				variant="outline"
				data-gsxui-slot-dropdown-menu-trigger
				aria-haspopup="menu"
				aria-expanded="false"
				class="self-start"
			>
				<icon.Languages/>
				<span lang={intl.Current(ctx).ID()} translate="no">{ intl.Current(ctx).Name() }</span>
			</ui.Button>
			<ui.DropdownMenuContent>
				<ui.DropdownMenuLabel>{ locales.ComponentsDropdownLabel }</ui.DropdownMenuLabel>
				<ui.DropdownMenuSeparator/>
				<ui.DropdownMenuRadioGroup value={intl.Current(ctx).ID()}>
					{ for _, l := range intl.Links(ctx) {
						<ui.DropdownMenuRadioItem checked={l.Current} value={l.Lang}>
							<a href={l.Href} lang={l.Lang} hreflang={l.Lang} translate="no" class="w-full">{ l.Name }</a>
						</ui.DropdownMenuRadioItem>
					} }
				</ui.DropdownMenuRadioGroup>
			</ui.DropdownMenuContent>
		</ui.DropdownMenu>
	</Demo>
}

component ScrollAreaDemo() {
	<Demo name="ui.ScrollArea" about={locales.ComponentsScrollAbout} wide={false}>
		<ui.ScrollArea class="h-64 rounded-md border">
			<ul class="flex flex-col p-2 text-sm">
				{ for _, c := range Sorted(ctx, "currency", Currencies) {
					<li class="flex justify-between gap-4 border-b py-1.5 last:border-0">
						<span data-intl-text>{ c.Name }</span>
						<span class="shrink-0 tabular-nums">{ 1 |> number("style", "currency", "currency", c.Code) }</span>
					</li>
				} }
			</ul>
		</ui.ScrollArea>
	</Demo>
}

component CollapsibleDemo() {
	<Demo name="ui.Collapsible" about={locales.ComponentsCollapsibleAbout} wide={false}>
		<ui.Collapsible class="flex flex-col gap-2">
			<ui.CollapsibleTrigger
				class="flex cursor-pointer items-center justify-between gap-4 rounded-md border px-3 py-2 text-sm font-medium"
			>
				<span>{ locales.ComponentsCollapsibleTrigger(len(NumberingSystems)) }</span>
				<span aria-hidden="true">
					<icon.ChevronsUpDown class="size-4"/>
				</span>
			</ui.CollapsibleTrigger>
			<NumberingRow id="latn"/>
			<ui.CollapsibleContent class="flex flex-col gap-2">
				{ for _, n := range NumberingSystems[1:] {
					<NumberingRow id={n}/>
				} }
			</ui.CollapsibleContent>
		</ui.Collapsible>
	</Demo>
}

// NumberingRow is a numbering system's id and its ten digits.
component NumberingRow(id string) {
	<div class="flex items-center justify-between gap-4 rounded-md border px-3 py-1.5 text-sm">
		<code translate="no">{ id }</code>
		<span class="tracking-widest">{ 1234567890 |> number("useGrouping", "false", "numberingSystem", id) }</span>
	</div>
}

component AccordionDemo() {
	<Demo name="ui.Accordion" about={locales.ComponentsAccordionAbout} wide={false}>
		<ui.Accordion name="components-facts">
			{ for i, f := range LocaleInfo() {
				<ui.AccordionItem name="components-facts" open={i == 0}>
					<ui.AccordionTrigger>{ f.Label }</ui.AccordionTrigger>
					<ui.AccordionContent>
						<code dir="ltr" class="text-xs break-all">{ f.V }</code>
					</ui.AccordionContent>
				</ui.AccordionItem>
			} }
		</ui.Accordion>
	</Demo>
}

component ListDemo() {
	<Demo name="ui.ToggleGroup" about={locales.ComponentsListAbout} wide={false}>
		<ui.ToggleGroup
			groupType="single"
			variant="outline"
			size="sm"
			spacing="0"
			aria-label={locales.ComponentsListLabel}
			class="self-start"
		>
			{ for _, c := range listTypes {
				<ui.ToggleGroupItem
					groupType="single"
					variant="outline"
					size="sm"
					spacing="0"
					pressed={c.ID == "conjunction"}
					value={c.ID}
					hx-on:click=js`document.querySelector("#list-result [data-intl]").dataset.opts = @{intl.Options("type", c.ID, "style", "long")}`
				>
					{ c.Label }
				</ui.ToggleGroupItem>
			} }
		</ui.ToggleGroup>
		<output id="list-result" class="text-lg">{ listItems(ctx) |> list("type", "conjunction", "style", "long") }</output>
	</Demo>
}

component TextareaDemo() {
	<Demo name="ui.Textarea" about={locales.ComponentsTextareaAbout} wide={false}>
		<form
			hx-post={"/components/count" |> localize}
			hx-trigger="input delay:200ms"
			hx-target="#count-result"
			class="flex flex-col gap-2"
		>
			<ui.Label for="components-text">{ locales.ComponentsTextareaLabel }</ui.Label>
			<ui.Textarea id="components-text" name="text" rows="4" value={locales.ComponentsTextareaSample |> text}/>
			<output id="count-result" class="text-sm text-muted-foreground">
				<TextCounts c={CountText(ctx, intl.Text(ctx, locales.ComponentsTextareaSample))}/>
			</output>
		</form>
	</Demo>
}

// TextCounts answers POST /components/count: how many characters, words and sentences, in plural forms.
component TextCounts(c Counts) {
	{ locales.ComponentsTextareaCounts(c.Graphemes, c.Words, c.Sentences) }
}

component AvatarDemo() {
	<Demo name="ui.Avatar" about={locales.ComponentsAvatarAbout} wide={false}>
		<ul class="flex flex-wrap gap-4">
			{ for _, p := range People {
				<li class="flex w-20 flex-col items-center gap-1 text-center">
					<ui.Avatar data-size="lg">
						<ui.AvatarFallback>{ Initials(ctx, p) }</ui.AvatarFallback>
					</ui.Avatar>
					<span class="text-xs" translate="no" dir="auto">{ p }</span>
				</li>
			} }
		</ul>
	</Demo>
}

component MenubarDemo() {
	<Demo name="ui.Menubar" about={locales.ComponentsMenubarAbout} wide={false}>
		<ui.Menubar class="self-start">
			<ui.MenubarMenu>
				<ui.MenubarTrigger>{ locales.ComponentsKindDate }</ui.MenubarTrigger>
				<ui.MenubarContent>
					{ for _, c := range dateStyles {
						<ui.MenubarItem
							hx-on:click=js`document.querySelector("#menubar-date [data-intl]").dataset.opts = @{intl.Options("dateStyle", c.ID, "timeStyle", "short")}`
						>
							{ c.Label }
						</ui.MenubarItem>
					} }
				</ui.MenubarContent>
			</ui.MenubarMenu>
			<ui.MenubarMenu>
				<ui.MenubarTrigger>{ locales.ComponentsKindNumber }</ui.MenubarTrigger>
				<ui.MenubarContent>
					{ for _, c := range notations {
						<ui.MenubarItem
							hx-on:click=js`document.querySelector("#menubar-number [data-intl]").dataset.opts = @{intl.Options("notation", c.ID)}`
						>
							{ c.Label }
						</ui.MenubarItem>
					} }
				</ui.MenubarContent>
			</ui.MenubarMenu>
		</ui.Menubar>
		<dl class="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-sm">
			<dt class="text-muted-foreground">{ locales.ComponentsKindDate }</dt>
			<dd id="menubar-date" class="font-medium">
				{ componentsDay |> date("dateStyle", "full", "timeStyle", "short") }
			</dd>
			<dt class="text-muted-foreground">{ locales.ComponentsKindNumber }</dt>
			<dd id="menubar-number" class="font-medium">{ 1234567.891 |> number("notation", "standard") }</dd>
		</dl>
	</Demo>
}

component ContextMenuDemo() {
	<Demo name="ui.ContextMenu" about={locales.ComponentsContextAbout} wide={false}>
		<ui.ContextMenu>
			<ui.ContextMenuTrigger
				class="flex min-h-28 items-center justify-center rounded-md border border-dashed p-4 text-center text-sm"
			>
				<span id="context-result" class="font-medium">{ componentsDay |> date("dateStyle", "full", "timeStyle", "short") }</span>
			</ui.ContextMenuTrigger>
			<ui.ContextMenuContent class="w-52">
				<ui.ContextMenuLabel>{ locales.ComponentsContextLabel }</ui.ContextMenuLabel>
				<ui.ContextMenuSeparator/>
				{ for _, c := range dateStyles {
					<ui.ContextMenuItem
						hx-on:click=js`document.querySelector("#context-result [data-intl]").dataset.opts = @{intl.Options("dateStyle", c.ID, "timeStyle", "short")}`
					>
						{ c.Label }
					</ui.ContextMenuItem>
				} }
			</ui.ContextMenuContent>
		</ui.ContextMenu>
		<p class="text-sm text-muted-foreground">{ locales.ComponentsContextHint }</p>
	</Demo>
}

component OverlayDemos() {
	<Demo name="ui.AlertDialog" about={locales.ComponentsAlertDialogAbout} wide={false}>
		<ui.AlertDialog>
			<ui.Button
				variant="outline"
				data-gsxui-slot-alert-dialog-trigger
				aria-haspopup="dialog"
				aria-expanded="false"
				class="self-start"
			>
				<icon.Trash2/>
				{ locales.ComponentsAlertDialogOpen(3) }
			</ui.Button>
			<ui.AlertDialogContent>
				<ui.AlertDialogHeader>
					<ui.AlertDialogTitle>{ locales.ComponentsAlertDialogTitle(3) }</ui.AlertDialogTitle>
					<ui.AlertDialogDescription>
						{ locales.ComponentsAlertDialogDescription(3, intl.Since(componentsDay)) }
					</ui.AlertDialogDescription>
				</ui.AlertDialogHeader>
				<ui.AlertDialogFooter>
					<ui.AlertDialogCancel>{ locales.ComponentsAlertDialogCancel }</ui.AlertDialogCancel>
					<ui.AlertDialogAction>{ locales.ComponentsAlertDialogConfirm }</ui.AlertDialogAction>
				</ui.AlertDialogFooter>
			</ui.AlertDialogContent>
		</ui.AlertDialog>
	</Demo>
}

component LoadingDemo() {
	<Demo name="ui.Skeleton · ui.Spinner" about={locales.ComponentsLoadingAbout} wide={false}>
		<div
			hx-get={"/fragments/components-week" |> localize}
			hx-trigger="load delay:600ms"
			hx-swap="outerHTML"
			class="flex flex-col gap-2"
		>
			<div class="flex items-center gap-2 text-sm text-muted-foreground">
				<ui.Spinner/>
				{ locales.ComponentsLoading }
			</div>
			<ui.Skeleton class="h-4 w-3/4"/>
			<ui.Skeleton class="h-4 w-1/2"/>
			<ui.Skeleton class="h-4 w-2/3"/>
		</div>
	</Demo>
}

// Week answers GET /fragments/components-week: the week from the page's day, starting where the locale's week
// starts (Intl.Locale week info), each day named by Intl.
component Week() {
	<ul class="flex flex-col gap-1 text-sm">
		{ for _, d := range WeekOf(ctx, componentsDay) {
			<li class={ "flex justify-between gap-4", "font-semibold": d.Equal(componentsDay) }>
				{ d |> date("weekday", "long", "day", "numeric", "month", "long", "timeZone", "UTC") }
			</li>
		} }
	</ul>
}

component KbdDemo() {
	<Demo name="ui.Kbd" about={locales.ComponentsKbdAbout} wide={false}>
		<ul class="flex flex-col gap-2 text-sm">
			<li class="flex items-center justify-between gap-4">
				<span>{ locales.ComponentsKbdCommand }</span>
				<ui.KbdGroup>
					<ui.Kbd>{ locales.ComponentsKbdCtrl }</ui.Kbd>
					<ui.Kbd translate="no">K</ui.Kbd>
				</ui.KbdGroup>
			</li>
			<li class="flex items-center justify-between gap-4">
				<span>{ locales.ComponentsKbdSidebar }</span>
				<ui.KbdGroup>
					<ui.Kbd>{ locales.ComponentsKbdCtrl }</ui.Kbd>
					<ui.Kbd translate="no">B</ui.Kbd>
				</ui.KbdGroup>
			</li>
			<li class="flex items-center justify-between gap-4">
				<span>{ locales.ComponentsKbdClose }</span>
				<ui.Kbd>{ locales.ComponentsKbdEscape }</ui.Kbd>
			</li>
		</ul>
	</Demo>
}

component SidebarDemo() {
	<Demo name="ui.Sidebar" about={locales.ComponentsSidebarAbout} wide={true}>
		<iframe
			src={"/components/sidebar" |> localize}
			title={locales.ComponentsSidebarFrame}
			data-site-isolated-preview
			loading="lazy"
			class="h-96 w-full rounded-lg border"
		></iframe>
	</Demo>
}

// ComponentsSidebar is /components/sidebar, the Sidebar's own document (it owns the viewport, so gsxui's site shows it
// in an iframe too): the page's sections as its menu, in the request's language.
component ComponentsSidebar() {
	<!DOCTYPE html>
	<html { intl.HTML(ctx)... }>
		<head>
			<meta charset="utf-8"/>
			<meta name="viewport" content="width=device-width, initial-scale=1"/>
			<title>{ locales.ComponentsSidebarFrame }</title>
			<documentAssets/>
		</head>
		<body class="bg-background font-sans text-foreground antialiased">
			<ui.SidebarProvider open={true}>
				<ui.Sidebar open={true} collapsible="icon" aria-label={locales.ComponentsSidebarFrame}>
					<ui.SidebarHeader>
						<div class="px-2 py-1 text-sm font-semibold" translate="no">go-htmx4</div>
					</ui.SidebarHeader>
					<ui.SidebarContent>
						<ui.SidebarGroup>
							<ui.SidebarGroupLabel>{ locales.ComponentsNavSections }</ui.SidebarGroupLabel>
							<ui.SidebarMenu>
								{ for i, s := range componentSections {
									<ui.SidebarMenuItem>
										<ui.SidebarMenuButton
											isActive={i == 0}
											tooltip={s.Title |> text}
											href={f`/components#@{s.ID}` |> localize}
											target="_top"
										>
											<icon.Hash/>
											<span>{ s.Title }</span>
										</ui.SidebarMenuButton>
									</ui.SidebarMenuItem>
								} }
							</ui.SidebarMenu>
						</ui.SidebarGroup>
					</ui.SidebarContent>
					<ui.SidebarRail/>
				</ui.Sidebar>
				<ui.SidebarInset>
					<header class="flex h-12 items-center gap-2 border-b px-4">
						<ui.SidebarTrigger/>
						<span class="text-sm font-medium">{ locales.ComponentsTitle }</span>
					</header>
					<div class="flex flex-col gap-2 p-4 text-sm">
						<p>{ locales.ComponentsSidebarBody }</p>
						<p class="text-muted-foreground">{ time.Now() |> date("dateStyle", "full") }</p>
					</div>
				</ui.SidebarInset>
			</ui.SidebarProvider>
		</body>
	</html>
}
