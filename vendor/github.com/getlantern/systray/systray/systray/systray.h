#pragma once

extern "C" {
	__declspec(dllexport) int __cdecl nativeLoop(void (*systray_ready)(int ignored), void (*_systray_menu_item_selected)(int menu_id));

	__declspec(dllexport) void __cdecl setIcon(const wchar_t* iconFile);
	__declspec(dllexport) void __cdecl setTooltip(const wchar_t* tooltip);
	__declspec(dllexport) void __cdecl add_or_update_menu_item(int menuId, wchar_t* title, wchar_t* tooltip, short disabled, short checked);
	__declspec(dllexport) void __cdecl quit();
}