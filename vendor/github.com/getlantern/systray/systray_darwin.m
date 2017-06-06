#import <Cocoa/Cocoa.h>
#include "systray.h"

@interface MenuItem : NSObject
{
  @public
    NSNumber* menuId;
    NSString* title;
    NSString* tooltip;
    short disabled;
    short checked;
}
-(id) initWithId: (int)theMenuId
       withTitle: (const char*)theTitle
     withTooltip: (const char*)theTooltip
    withDisabled: (short)theDisabled
     withChecked: (short)theChecked;
     @end
     @implementation MenuItem
     -(id) initWithId: (int)theMenuId
            withTitle: (const char*)theTitle
          withTooltip: (const char*)theTooltip
         withDisabled: (short)theDisabled
          withChecked: (short)theChecked
{
  menuId = [NSNumber numberWithInt:theMenuId];
  title = [[NSString alloc] initWithCString:theTitle
                                   encoding:NSUTF8StringEncoding];
  tooltip = [[NSString alloc] initWithCString:theTooltip
                                     encoding:NSUTF8StringEncoding];
  disabled = theDisabled;
  checked = theChecked;
  return self;
}
@end

@interface AppDelegate: NSObject <NSApplicationDelegate>
  - (void) add_or_update_menu_item:(MenuItem*) item;
  - (IBAction)menuHandler:(id)sender;
  @property (assign) IBOutlet NSWindow *window;
  @end

  @implementation AppDelegate
{
  NSStatusItem *statusItem;
  NSMenu *menu;
  NSCondition* cond;
}

@synthesize window = _window;

- (void)applicationDidFinishLaunching:(NSNotification *)aNotification
{
  self->statusItem = [[NSStatusBar systemStatusBar] statusItemWithLength:NSVariableStatusItemLength];
  self->menu = [[NSMenu alloc] init];
  [self->menu setAutoenablesItems: FALSE];
  [self->statusItem setMenu:self->menu];
  systray_ready();
}

- (void)setIcon:(NSImage *)image {
  [statusItem setImage:image];
}

- (void)setTitle:(NSString *)title {
  [statusItem setTitle:title];
}

- (void)setTooltip:(NSString *)tooltip {
  [statusItem setToolTip:tooltip];
}

- (IBAction)menuHandler:(id)sender
{
  NSNumber* menuId = [sender representedObject];
  systray_menu_item_selected(menuId.intValue);
}

- (void) add_or_update_menu_item:(MenuItem*) item
{
  NSMenuItem* menuItem;
  int existedMenuIndex = [menu indexOfItemWithRepresentedObject: item->menuId];
  if (existedMenuIndex == -1) {
    menuItem = [menu addItemWithTitle:item->title action:@selector(menuHandler:) keyEquivalent:@""];
    [menuItem setTarget:self];
    [menuItem setRepresentedObject: item->menuId];

  }
  else {
    menuItem = [menu itemAtIndex: existedMenuIndex];
    [menuItem setTitle:item->title];
  }
  [menuItem setToolTip:item->tooltip];
  if (item->disabled == 1) {
    [menuItem setEnabled:FALSE];
  } else {
    [menuItem setEnabled:TRUE];
  }
  if (item->checked == 1) {
    [menuItem setState:NSOnState];
  } else {
    [menuItem setState:NSOffState];
  }
}

- (void) quit
{
  [[NSStatusBar systemStatusBar] removeStatusItem: statusItem];
}

@end

int nativeLoop(void) {
  AppDelegate *delegate = [[AppDelegate alloc] init];
  [[NSApplication sharedApplication] setDelegate:delegate];
  [NSApp run];
  return EXIT_SUCCESS;
}

void runInMainThread(SEL method, id object) {
  [(AppDelegate*)[NSApp delegate]
    performSelectorOnMainThread:method
                     withObject:object
                  waitUntilDone: YES];
}

void setIcon(const char* iconBytes, int length) {
  NSData* buffer = [NSData dataWithBytes: iconBytes length:length];
  NSImage *image = [[NSImage alloc] initWithData:buffer];
  runInMainThread(@selector(setIcon:), (id)image);
}

void setTitle(char* ctitle) {
  NSString* title = [[NSString alloc] initWithCString:ctitle
                                             encoding:NSUTF8StringEncoding];
  free(ctitle);
  runInMainThread(@selector(setTitle:), (id)title);
}

void setTooltip(char* ctooltip) {
  NSString* tooltip = [[NSString alloc] initWithCString:ctooltip
                                               encoding:NSUTF8StringEncoding];
  free(ctooltip);
  runInMainThread(@selector(setTooltip:), (id)tooltip);
}

void add_or_update_menu_item(int menuId, char* title, char* tooltip, short disabled, short checked) {
  MenuItem* item = [[MenuItem alloc] initWithId: menuId withTitle: title withTooltip: tooltip withDisabled: disabled withChecked: checked];
  free(title);
  free(tooltip);
  runInMainThread(@selector(add_or_update_menu_item:), (id)item);
}

void quit() {
  runInMainThread(@selector(quit), nil);
}
