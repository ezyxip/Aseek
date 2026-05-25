Name:       ru.pmifi.Aseek
Summary:    Semantic search
Version:    0.1
Release:    1
License:    BSD-3-Clause
URL:        https://auroraos.ru
Source0:    %{name}-%{version}.tar.bz2

Requires:   sailfishsilica-qt5 >= 0.10.9
BuildRequires:  pkgconfig(auroraapp)
BuildRequires:  pkgconfig(Qt5Core)
BuildRequires:  pkgconfig(Qt5Qml)
BuildRequires:  pkgconfig(Qt5Quick)

%description
Semantic search

%prep
%autosetup

%build
%cmake -GNinja
%ninja_build

%install
%ninja_install

# Remove files installed by llama.cpp subproject that we don't package
rm -f %{buildroot}%{_bindir}/llama-server
rm -rf %{buildroot}%{_includedir}/ggml*.h
rm -rf %{buildroot}%{_includedir}/llama*.h
rm -rf %{buildroot}%{_includedir}/gguf.h
rm -rf %{buildroot}%{_includedir}/mtmd*.h
rm -rf %{buildroot}%{_libdir}/cmake/ggml
rm -rf %{buildroot}%{_libdir}/cmake/llama
rm -f %{buildroot}%{_libdir}/libggml*.a
rm -f %{buildroot}%{_libdir}/libllama*.a
rm -f %{buildroot}%{_libdir}/libmtmd.a
rm -f %{buildroot}%{_libdir}/pkgconfig/llama.pc

%files
%defattr(-,root,root,-)
%{_bindir}/%{name}
%defattr(644,root,root,-)
%{_datadir}/%{name}
%{_datadir}/applications/%{name}.desktop
%{_datadir}/icons/hicolor/*/apps/%{name}.png
%defattr(755,root,root,-)
%dir %{_libexecdir}/%{name}
%{_libexecdir}/%{name}/llama-server
%{_libexecdir}/%{name}/aseek-orchestrator
